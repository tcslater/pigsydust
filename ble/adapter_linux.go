//go:build linux

package ble

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/tcslater/pigsydust"
)

// linuxAdapter talks to BlueZ (org.bluez) over the system D-Bus. It picks the
// first available adapter (typically /org/bluez/hci0).
type linuxAdapter struct {
	bus         *dbus.Conn
	adapterPath dbus.ObjectPath
}

// linuxAddress satisfies Address for BlueZ. Only the MAC is used in logs;
// the object path is what Device1.Connect takes.
type linuxAddress struct {
	path dbus.ObjectPath
	mac  string // "AA:BB:CC:DD:EE:FF"
}

func (l linuxAddress) String() string { return l.mac }

// NewAdapter connects to the system bus and finds the first BlueZ adapter.
func NewAdapter() (*Adapter, error) {
	bus, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("ble: connecting to system bus: %w", err)
	}
	path, err := findAdapter(bus)
	if err != nil {
		return nil, err
	}
	// Power on the adapter if it isn't already.
	if err := setProperty(bus, path, bluezAdapterIface, "Powered", dbus.MakeVariant(true)); err != nil {
		// Non-fatal: many adapters are already powered; SetProperty can
		// return a permission error when the service has already powered
		// the HCI.
		log.Printf("ble: warning: setting Powered=true: %v", err)
	}
	return &Adapter{impl: &linuxAdapter{bus: bus, adapterPath: path}}, nil
}

// findAdapter walks ObjectManager looking for the first Adapter1.
func findAdapter(bus *dbus.Conn) (dbus.ObjectPath, error) {
	objs, err := getManagedObjects(bus)
	if err != nil {
		return "", err
	}
	for path, ifs := range objs {
		if _, ok := ifs[bluezAdapterIface]; ok {
			return path, nil
		}
	}
	return "", errors.New("ble: no BlueZ adapter found (is bluetoothd running?)")
}

func getManagedObjects(bus *dbus.Conn) (map[dbus.ObjectPath]map[string]map[string]dbus.Variant, error) {
	obj := bus.Object(bluezBusName, bluezRootPath)
	var result map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := obj.Call(dbusObjectManager+".GetManagedObjects", 0).Store(&result); err != nil {
		return nil, fmt.Errorf("ble: GetManagedObjects: %w", err)
	}
	return result, nil
}

func setProperty(bus *dbus.Conn, path dbus.ObjectPath, iface, name string, value dbus.Variant) error {
	obj := bus.Object(bluezBusName, path)
	return obj.Call(dbusProperties+".Set", 0, iface, name, value).Err
}

func getProperty(bus *dbus.Conn, path dbus.ObjectPath, iface, name string) (dbus.Variant, error) {
	obj := bus.Object(bluezBusName, path)
	var v dbus.Variant
	err := obj.Call(dbusProperties+".Get", 0, iface, name).Store(&v)
	return v, err
}

// connect implements adapterImpl.connect on Linux.
func (a *linuxAdapter) connect(ctx context.Context, adv pigsydust.AdvertisementData, addr Address) (*Connection, error) {
	la, ok := addr.(linuxAddress)
	if !ok {
		return nil, fmt.Errorf("ble: unexpected address type %T", addr)
	}

	dev := a.bus.Object(bluezBusName, la.path)
	if err := dev.CallWithContext(ctx, bluezDeviceIface+".Connect", 0).Err; err != nil {
		return nil, fmt.Errorf("ble: Device1.Connect: %w", err)
	}

	// Wait for ServicesResolved before enumerating characteristics.
	if err := waitServicesResolved(ctx, a.bus, la.path, 10*time.Second); err != nil {
		dev.CallWithContext(ctx, bluezDeviceIface+".Disconnect", 0)
		return nil, err
	}

	// Walk ObjectManager for this device's GATT characteristics.
	objs, err := getManagedObjects(a.bus)
	if err != nil {
		dev.CallWithContext(ctx, bluezDeviceIface+".Disconnect", 0)
		return nil, err
	}

	charByUUID := make(map[UUID]dbus.ObjectPath)
	prefix := string(la.path) + "/"
	for path, ifs := range objs {
		if !strings.HasPrefix(string(path), prefix) {
			continue
		}
		charProps, ok := ifs[bluezGattCharIface]
		if !ok {
			continue
		}
		uuidV, ok := charProps["UUID"]
		if !ok {
			continue
		}
		uuidStr, _ := uuidV.Value().(string)
		u, err := parseUUIDString(uuidStr)
		if err != nil {
			continue
		}
		charByUUID[u] = path
	}

	conn := &Connection{gwMAC: adv.MAC}
	closed := false
	conn.closer = func() error {
		if closed {
			return nil
		}
		closed = true
		return dev.Call(bluezDeviceIface+".Disconnect", 0).Err
	}

	required := []struct {
		uuid UUID
		name string
		dst  *charIO
	}{
		{CharNotifyUUID, "CHAR_NOTIFY", &conn.charNotify},
		{CharCmdUUID, "CHAR_CMD", &conn.charCmd},
		{CharPairUUID, "CHAR_PAIR", &conn.charPair},
	}
	for _, r := range required {
		path, ok := charByUUID[r.uuid]
		if !ok {
			_ = conn.Close()
			return nil, fmt.Errorf("ble: %s characteristic not found", r.name)
		}
		*r.dst = newLinuxCharIO(a.bus, path)
	}

	return conn, nil
}

func (a *linuxAdapter) stopScan() error {
	obj := a.bus.Object(bluezBusName, a.adapterPath)
	return obj.Call(bluezAdapterIface+".StopDiscovery", 0).Err
}

// waitServicesResolved polls and subscribes to PropertiesChanged for the
// ServicesResolved boolean on the device path.
func waitServicesResolved(ctx context.Context, bus *dbus.Conn, path dbus.ObjectPath, timeout time.Duration) error {
	// Fast path: maybe already resolved.
	if v, err := getProperty(bus, path, bluezDeviceIface, "ServicesResolved"); err == nil {
		if b, ok := v.Value().(bool); ok && b {
			return nil
		}
	}

	matchRule := fmt.Sprintf("type='signal',sender='%s',interface='%s',member='PropertiesChanged',path='%s'",
		bluezBusName, dbusProperties, path)
	if err := bus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err; err != nil {
		return fmt.Errorf("ble: AddMatch ServicesResolved: %w", err)
	}
	defer bus.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, matchRule)

	ch := make(chan *dbus.Signal, 16)
	bus.Signal(ch)
	defer bus.RemoveSignal(ch)

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return ctx.Err()
		case <-deadline.C:
			return errors.New("ble: timed out waiting for ServicesResolved")
		case sig := <-ch:
			if sig == nil || sig.Path != path || sig.Name != propPropertiesChang {
				continue
			}
			if len(sig.Body) < 2 {
				continue
			}
			iface, _ := sig.Body[0].(string)
			if iface != bluezDeviceIface {
				continue
			}
			changed, _ := sig.Body[1].(map[string]dbus.Variant)
			if v, ok := changed["ServicesResolved"]; ok {
				if b, ok := v.Value().(bool); ok && b {
					return nil
				}
			}
		}
	}
}

// parseUUIDString parses a 8-4-4-4-12 UUID string into our UUID type.
func parseUUIDString(s string) (UUID, error) {
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return UUID{}, fmt.Errorf("ble: bad uuid %q", s)
	}
	var out UUID
	for i := 0; i < 16; i++ {
		var hi, lo byte
		if err := hexByte(s[2*i], &hi); err != nil {
			return UUID{}, err
		}
		if err := hexByte(s[2*i+1], &lo); err != nil {
			return UUID{}, err
		}
		out[i] = (hi << 4) | lo
	}
	return out, nil
}

func hexByte(c byte, out *byte) error {
	switch {
	case c >= '0' && c <= '9':
		*out = c - '0'
	case c >= 'a' && c <= 'f':
		*out = c - 'a' + 10
	case c >= 'A' && c <= 'F':
		*out = c - 'A' + 10
	default:
		return fmt.Errorf("ble: bad hex %q", c)
	}
	return nil
}
