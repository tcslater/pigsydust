package ble

import (
	"context"

	"github.com/tcslater/pigsydust"
)

// Address identifies a BLE peer in platform-specific terms. On Darwin this
// wraps a CoreBluetooth UUID; on Linux it wraps a BlueZ object path plus the
// real hardware MAC.
type Address interface {
	String() string
}

// ScanResult pairs parsed advertisement data with the raw BLE address
// needed for Adapter.Connect.
type ScanResult struct {
	Advertisement pigsydust.AdvertisementData
	Address       Address
	RSSI          int16
}

// Adapter wraps a platform-specific BLE host adapter.
type Adapter struct {
	impl adapterImpl
}

// Scan discovers Pixie mesh devices matching the given filter. See the
// platform adapter implementations for details on dedup/behaviour.
func (a *Adapter) Scan(ctx context.Context, filter pigsydust.ScanFilter) (<-chan ScanResult, error) {
	return a.impl.scan(ctx, filter)
}

// StopScan stops an active BLE scan.
func (a *Adapter) StopScan() error { return a.impl.stopScan() }

// Connect establishes a BLE connection to a previously scanned Pixie device
// and discovers the required GATT characteristics.
func (a *Adapter) Connect(ctx context.Context, adv pigsydust.AdvertisementData, addr Address) (*Connection, error) {
	return a.impl.connect(ctx, adv, addr)
}

// Connection holds a live GATT session with the three Pixie characteristics
// ready to read/write. Close disconnects.
type Connection struct {
	charNotify charIO
	charCmd    charIO
	charPair   charIO
	gwMAC      pigsydust.MACAddress
	closer     func() error
}

// Close disconnects from the BLE device.
func (c *Connection) Close() error {
	if c.closer == nil {
		return nil
	}
	return c.closer()
}

// adapterImpl is the platform-private hook behind Adapter. Tinygo and BlueZ
// implementations live in adapter_tinygo.go and adapter_linux.go respectively.
type adapterImpl interface {
	scan(ctx context.Context, filter pigsydust.ScanFilter) (<-chan ScanResult, error)
	stopScan() error
	connect(ctx context.Context, adv pigsydust.AdvertisementData, addr Address) (*Connection, error)
}

// charIO is the minimal read/write surface Transport needs on a GATT
// characteristic. Each platform plugs in its own implementation when it
// builds a Connection.
type charIO interface {
	// writeRequest issues an ATT Write Request (with response).
	writeRequest(ctx context.Context, data []byte) error
	// writeCommand issues an ATT Write Command (without response).
	writeCommand(ctx context.Context, data []byte) error
	// read performs a GATT Read on the characteristic.
	read(ctx context.Context) ([]byte, error)
	// subscribe arms notifications and returns a channel that delivers each
	// notification's raw bytes. The channel is closed when ctx is cancelled.
	subscribe(ctx context.Context) (<-chan []byte, error)
}
