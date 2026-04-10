package piggsydust

import "context"

// Transport abstracts BLE GATT operations for a connected mesh node.
// Implementations wrap platform-specific BLE libraries (CoreBluetooth,
// BlueZ, tinygo/bluetooth, go-ble/ble, etc.).
//
// The Telink mesh service UUID is 00010203-0405-0607-0809-0a0b0c0d1910.
// Characteristic UUIDs use suffixes 1911-1914 under this base.
//
// All methods must be safe for concurrent use.
// Context cancellation must be respected for all blocking operations.
type Transport interface {
	// WritePair writes data to CHAR_PAIR (UUID suffix 0x1914) using
	// ATT Write Request (with response). Used for the login handshake.
	WritePair(ctx context.Context, data []byte) error

	// ReadPair reads from CHAR_PAIR (UUID suffix 0x1914).
	// Used for the login response and heartbeat keepalive.
	ReadPair(ctx context.Context) ([]byte, error)

	// WriteCommand writes an encrypted packet to CHAR_CMD (UUID suffix 0x1912).
	WriteCommand(ctx context.Context, data []byte) error

	// SubscribeNotify subscribes to CHAR_NOTIFY (UUID suffix 0x1911) and
	// returns a channel delivering raw 20-byte notification packets.
	// The implementation must write 0x01 to enable notifications.
	// The channel is closed when the context is cancelled or the
	// connection drops.
	SubscribeNotify(ctx context.Context) (<-chan []byte, error)

	// GatewayMAC returns the 6-byte MAC address of the connected gateway
	// in standard order (AA:BB:CC:DD:EE:FF → index 0=AA, 5=FF).
	//
	// On macOS/iOS where the OS hides real MAC addresses, this should be
	// extracted from manufacturer data bytes 2-5 and/or the DIS Model
	// Number characteristic (0x2a24).
	GatewayMAC() MACAddress
}

// Scanner is an optional interface for BLE scanning. It is not required
// by [Client] but is provided as a convention for Transport implementations
// to follow.
type Scanner interface {
	// Scan discovers Pixie mesh devices matching the given filter.
	// Results are delivered on the returned channel, which is closed
	// when the context is cancelled.
	Scan(ctx context.Context, filter ScanFilter) (<-chan AdvertisementData, error)
}

// ScanFilter constrains which BLE advertisements are returned by [Scanner].
type ScanFilter struct {
	// MeshName filters by the advertised local name (e.g. "Smart Light").
	// Empty string matches any name.
	MeshName string

	// NetworkID filters by the mesh network ID from manufacturer data.
	// Zero matches any network.
	NetworkID uint32

	// GatewayOnly limits results to devices with DeviceType 0x47 (gateway).
	GatewayOnly bool
}
