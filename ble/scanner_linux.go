//go:build linux

package ble

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/tcslater/pigsydust"
)

// scan implements adapterImpl.scan on Linux using BlueZ object-manager signals.
func (a *linuxAdapter) scan(ctx context.Context, filter pigsydust.ScanFilter) (<-chan ScanResult, error) {
	// SetDiscoveryFilter to restrict results to LE advertisements matching
	// the Pixie service UUID 0xCDAB, which cuts the PropertiesChanged rate
	// meaningfully on busy environments.
	adapter := a.bus.Object(bluezBusName, a.adapterPath)
	serviceFilter := map[string]dbus.Variant{
		"Transport": dbus.MakeVariant("le"),
		"UUIDs":     dbus.MakeVariant([]string{ServiceUUID.String()}),
	}
	if err := adapter.CallWithContext(ctx, bluezAdapterIface+".SetDiscoveryFilter", 0, serviceFilter).Err; err != nil {
		return nil, fmt.Errorf("ble: SetDiscoveryFilter: %w", err)
	}

	// Subscribe to ObjectManager + PropertiesChanged signals for all BlueZ objects.
	rules := []string{
		fmt.Sprintf("type='signal',sender='%s',interface='%s',member='InterfacesAdded'", bluezBusName, dbusObjectManager),
		fmt.Sprintf("type='signal',sender='%s',interface='%s',member='PropertiesChanged'", bluezBusName, dbusProperties),
	}
	for _, r := range rules {
		if err := a.bus.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, r).Err; err != nil {
			return nil, fmt.Errorf("ble: AddMatch: %w", err)
		}
	}

	sigCh := make(chan *dbus.Signal, 128)
	a.bus.Signal(sigCh)

	out := make(chan ScanResult, 16)

	// Also drain existing objects (devices may already be cached from a prior scan).
	go func() {
		defer close(out)
		defer func() {
			for _, r := range rules {
				a.bus.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, r)
			}
			a.bus.RemoveSignal(sigCh)
		}()

		// Seed with currently-known devices.
		if objs, err := getManagedObjects(a.bus); err == nil {
			for path, ifs := range objs {
				if devProps, ok := ifs[bluezDeviceIface]; ok {
					maybeEmit(out, path, devProps, filter, ctx)
				}
			}
		}

		if err := adapter.CallWithContext(ctx, bluezAdapterIface+".StartDiscovery", 0).Err; err != nil {
			return
		}
		defer adapter.Call(bluezAdapterIface+".StopDiscovery", 0)

		props := make(map[dbus.ObjectPath]map[string]dbus.Variant)
		for {
			select {
			case <-ctx.Done():
				return
			case sig, ok := <-sigCh:
				if !ok {
					return
				}
				if sig == nil {
					continue
				}
				switch sig.Name {
				case propInterfacesAdded:
					if len(sig.Body) < 2 {
						continue
					}
					path, _ := sig.Body[0].(dbus.ObjectPath)
					ifs, _ := sig.Body[1].(map[string]map[string]dbus.Variant)
					devProps, ok := ifs[bluezDeviceIface]
					if !ok {
						continue
					}
					merged := mergeProps(props[path], devProps)
					props[path] = merged
					maybeEmit(out, path, merged, filter, ctx)
				case propPropertiesChang:
					if len(sig.Body) < 2 {
						continue
					}
					iface, _ := sig.Body[0].(string)
					if iface != bluezDeviceIface {
						continue
					}
					changed, _ := sig.Body[1].(map[string]dbus.Variant)
					merged := mergeProps(props[sig.Path], changed)
					props[sig.Path] = merged
					maybeEmit(out, sig.Path, merged, filter, ctx)
				}
			}
		}
	}()

	return out, nil
}

func mergeProps(a, b map[string]dbus.Variant) map[string]dbus.Variant {
	out := make(map[string]dbus.Variant, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// maybeEmit tries to build a ScanResult from a device's accumulated properties.
func maybeEmit(out chan<- ScanResult, path dbus.ObjectPath, props map[string]dbus.Variant, filter pigsydust.ScanFilter, ctx context.Context) {
	mfgV, ok := props["ManufacturerData"]
	if !ok {
		return
	}
	mfg, ok := mfgV.Value().(map[uint16]dbus.Variant)
	if !ok {
		return
	}
	raw, ok := mfg[ManufacturerIDSkytone]
	if !ok {
		return
	}
	mfgData, ok := raw.Value().([]byte)
	if !ok {
		return
	}

	var name string
	if v, ok := props["Name"]; ok {
		name, _ = v.Value().(string)
	}
	if filter.MeshName != "" && name != filter.MeshName {
		return
	}

	adv, err := pigsydust.ParseManufacturerData(ManufacturerIDSkytone, mfgData)
	if err != nil {
		return
	}
	adv.MeshName = name

	if filter.NetworkID != 0 && adv.NetworkID != filter.NetworkID {
		return
	}
	if filter.GatewayOnly && adv.DeviceType != pigsydust.DeviceTypeGateway {
		return
	}

	// Prefer the real MAC from Device1.Address, falling back to the object
	// path suffix (/dev_AA_BB_CC_DD_EE_FF) which encodes the MAC directly.
	mac := ""
	if v, ok := props["Address"]; ok {
		mac, _ = v.Value().(string)
	}
	if mac == "" {
		mac = macFromPath(path)
	}

	var rssi int16
	if v, ok := props["RSSI"]; ok {
		if r, ok := v.Value().(int16); ok {
			rssi = r
		}
	}

	select {
	case out <- ScanResult{
		Advertisement: adv,
		Address:       linuxAddress{path: path, mac: mac},
		RSSI:          rssi,
	}:
	case <-ctx.Done():
	}
}

// macFromPath extracts "AA:BB:CC:DD:EE:FF" from a BlueZ device path
// like "/org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF".
func macFromPath(p dbus.ObjectPath) string {
	s := string(p)
	i := strings.LastIndex(s, "/dev_")
	if i < 0 {
		return ""
	}
	return strings.ReplaceAll(s[i+5:], "_", ":")
}
