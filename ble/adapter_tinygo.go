//go:build !linux

package ble

import (
	"context"
	"errors"
	"fmt"

	"github.com/tcslater/pigsydust"
	"tinygo.org/x/bluetooth"
)

// tinygoAdapter implements adapterImpl on top of tinygo.org/x/bluetooth. This
// is the Darwin and Windows path; Linux has a dedicated godbus implementation.
type tinygoAdapter struct {
	bt *bluetooth.Adapter
}

// NewAdapter creates and enables the default BLE adapter.
func NewAdapter() (*Adapter, error) {
	a := bluetooth.DefaultAdapter
	if err := a.Enable(); err != nil {
		return nil, fmt.Errorf("ble: enabling adapter: %w", err)
	}
	return &Adapter{impl: &tinygoAdapter{bt: a}}, nil
}

// tinygoAddress wraps a bluetooth.Address as our platform-neutral Address.
type tinygoAddress struct{ a bluetooth.Address }

func (t tinygoAddress) String() string { return t.a.String() }

func toTinygoUUID(u UUID) bluetooth.UUID {
	return bluetooth.NewUUID(u)
}

func fromTinygoUUID(u bluetooth.UUID) UUID {
	b := u.Bytes()
	var out UUID
	copy(out[:], b[:])
	return out
}

func (a *tinygoAdapter) connect(ctx context.Context, adv pigsydust.AdvertisementData, addr Address) (*Connection, error) {
	ta, ok := addr.(tinygoAddress)
	if !ok {
		return nil, fmt.Errorf("ble: unexpected address type %T", addr)
	}
	device, err := a.bt.Connect(ta.a, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, fmt.Errorf("ble: connecting: %w", err)
	}

	// Discover the Telink mesh service.
	svcs, err := device.DiscoverServices([]bluetooth.UUID{toTinygoUUID(MeshServiceUUID)})
	if err != nil {
		device.Disconnect()
		return nil, fmt.Errorf("ble: discovering mesh service: %w", err)
	}
	if len(svcs) == 0 {
		device.Disconnect()
		return nil, errors.New("ble: mesh service not found")
	}

	chars, err := svcs[0].DiscoverCharacteristics([]bluetooth.UUID{
		toTinygoUUID(CharNotifyUUID),
		toTinygoUUID(CharCmdUUID),
		toTinygoUUID(CharPairUUID),
	})
	if err != nil {
		device.Disconnect()
		return nil, fmt.Errorf("ble: discovering characteristics: %w", err)
	}

	conn := &Connection{gwMAC: adv.MAC}
	dev := device
	conn.closer = func() error { return dev.Disconnect() }

	for _, ch := range chars {
		io := &tinygoCharIO{ch: ch}
		switch fromTinygoUUID(ch.UUID()) {
		case CharNotifyUUID:
			conn.charNotify = io
		case CharCmdUUID:
			conn.charCmd = io
		case CharPairUUID:
			conn.charPair = io
		}
	}

	// enrichMACFromDIS: on macOS the OS randomises BLE addresses, so we read
	// the real MAC from the DIS model number characteristic if present. A
	// missing/unreadable DIS is non-fatal — the scan already parsed a MAC.
	_ = enrichMACFromDIS(device, conn)

	return conn, nil
}

// enrichMACFromDIS reads the Device Information Service Model Number
// characteristic to extract the full MAC address string.
func enrichMACFromDIS(device bluetooth.Device, conn *Connection) error {
	svcs, err := device.DiscoverServices([]bluetooth.UUID{toTinygoUUID(disServiceUUID)})
	if err != nil || len(svcs) == 0 {
		return fmt.Errorf("ble: DIS service not found")
	}

	chars, err := svcs[0].DiscoverCharacteristics([]bluetooth.UUID{toTinygoUUID(disModelNumberUUID)})
	if err != nil || len(chars) == 0 {
		return fmt.Errorf("ble: DIS model number not found")
	}

	buf := make([]byte, 32)
	n, err := chars[0].Read(buf)
	if err != nil {
		return fmt.Errorf("ble: reading DIS model number: %w", err)
	}

	mac, err := pigsydust.ParseMAC(string(buf[:n]))
	if err != nil {
		return err
	}

	conn.gwMAC = mac
	return nil
}
