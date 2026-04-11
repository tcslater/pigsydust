package ble

import (
	"context"
	"errors"
	"fmt"

	"github.com/tcslater/pigsydust"
	"tinygo.org/x/bluetooth"
)

// Well-known UUIDs for the Telink mesh GATT service and characteristics.
var (
	// ServiceUUID is the 16-bit Pixie advertisement service UUID.
	ServiceUUID = bluetooth.New16BitUUID(0xCDAB)

	// MeshServiceUUID is the primary Telink mesh GATT service.
	MeshServiceUUID = bluetooth.NewUUID([16]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x10,
	})

	// CharNotifyUUID is CHAR_NOTIFY — subscribe for encrypted notifications.
	CharNotifyUUID = bluetooth.NewUUID([16]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x11,
	})

	// CharCmdUUID is CHAR_CMD — write encrypted command packets.
	CharCmdUUID = bluetooth.NewUUID([16]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x12,
	})

	// CharOTAUUID is CHAR_OTA — OTA/config (not used in normal operation).
	CharOTAUUID = bluetooth.NewUUID([16]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x13,
	})

	// CharPairUUID is CHAR_PAIR — login handshake and heartbeat.
	CharPairUUID = bluetooth.NewUUID([16]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x14,
	})

	// DIS UUIDs for MAC address extraction.
	disServiceUUID    = bluetooth.New16BitUUID(0x180a)
	disModelNumberUID = bluetooth.New16BitUUID(0x2a24)
)

// ManufacturerIDSkytone is the BLE manufacturer company ID for Pixie devices.
const ManufacturerIDSkytone = 0x0211

// Adapter wraps a [bluetooth.Adapter] and provides Pixie-specific operations.
type Adapter struct {
	bt *bluetooth.Adapter
}

// NewAdapter creates and enables the default BLE adapter.
func NewAdapter() (*Adapter, error) {
	a := bluetooth.DefaultAdapter
	if err := a.Enable(); err != nil {
		return nil, fmt.Errorf("ble: enabling adapter: %w", err)
	}
	return &Adapter{bt: a}, nil
}

// Connection holds a connected BLE device and its discovered GATT characteristics.
type Connection struct {
	device bluetooth.Device
	gwMAC  pigsydust.MACAddress

	charNotify bluetooth.DeviceCharacteristic
	charCmd    bluetooth.DeviceCharacteristic
	charPair   bluetooth.DeviceCharacteristic
}

// Close disconnects from the BLE device.
func (c *Connection) Close() error {
	return c.device.Disconnect()
}

// Connect establishes a BLE connection to a previously scanned Pixie device
// and discovers the required GATT characteristics.
//
// The adv parameter should come from a prior [Adapter.Scan] call. The MAC
// address is extracted from the advertisement's manufacturer data.
func (a *Adapter) Connect(ctx context.Context, adv pigsydust.AdvertisementData, addr bluetooth.Address) (*Connection, error) {
	device, err := a.bt.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, fmt.Errorf("ble: connecting: %w", err)
	}

	conn := &Connection{
		device: device,
		gwMAC:  adv.MAC,
	}

	// Discover the Telink mesh service.
	svcs, err := device.DiscoverServices([]bluetooth.UUID{MeshServiceUUID})
	if err != nil {
		device.Disconnect()
		return nil, fmt.Errorf("ble: discovering mesh service: %w", err)
	}
	if len(svcs) == 0 {
		device.Disconnect()
		return nil, errors.New("ble: mesh service not found")
	}

	// Discover all four characteristics.
	chars, err := svcs[0].DiscoverCharacteristics([]bluetooth.UUID{
		CharNotifyUUID, CharCmdUUID, CharPairUUID,
	})
	if err != nil {
		device.Disconnect()
		return nil, fmt.Errorf("ble: discovering characteristics: %w", err)
	}

	for _, ch := range chars {
		switch ch.UUID() {
		case CharNotifyUUID:
			conn.charNotify = ch
		case CharCmdUUID:
			conn.charCmd = ch
		case CharPairUUID:
			conn.charPair = ch
		}
	}

	// Try to extract full MAC from DIS Model Number if partial.
	if err := conn.enrichMACFromDIS(device); err != nil {
		// Non-fatal — we may already have enough from manufacturer data.
	}

	return conn, nil
}

// enrichMACFromDIS reads the Device Information Service Model Number
// characteristic to extract the full MAC address string (e.g. "AA:BB:CC:DD:EE:FF").
// This is particularly useful on macOS/iOS where the OS hides real MAC addresses.
func (c *Connection) enrichMACFromDIS(device bluetooth.Device) error {
	svcs, err := device.DiscoverServices([]bluetooth.UUID{disServiceUUID})
	if err != nil || len(svcs) == 0 {
		return fmt.Errorf("ble: DIS service not found")
	}

	chars, err := svcs[0].DiscoverCharacteristics([]bluetooth.UUID{disModelNumberUID})
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

	c.gwMAC = mac
	return nil
}
