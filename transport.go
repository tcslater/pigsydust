package pigsydust

import "context"

// Transport is the GATT-level interface a [Client] uses to talk to a
// connected Pixie mesh node. Implementations are provided out-of-tree (e.g.
// the ble/ module) so that the core library stays BLE-stack-agnostic.
//
// Call ordering during a session:
//
//  1. WritePair + ReadPair — login handshake (see crypto.BuildLoginRequest /
//     crypto.ParseLoginResponse).
//  2. SubscribeNotify — subscribe to encrypted notification packets.
//  3. WriteCommand — send encrypted command packets (many, throughout the
//     session).
//  4. ReadPair — periodic keepalive (every < 30s).
//
// Implementations must be safe for concurrent use by a single Client. Writes
// may interleave from different goroutines (e.g. heartbeat + control) so the
// transport should serialise them internally.
type Transport interface {
	// WritePair writes to CHAR_PAIR (0x1914) with an ATT Write Request
	// (response required). Used only during login.
	WritePair(ctx context.Context, data []byte) error

	// ReadPair reads from CHAR_PAIR (0x1914). Used for the login response
	// and as the periodic keepalive.
	ReadPair(ctx context.Context) ([]byte, error)

	// WriteCommand writes an encrypted command packet to CHAR_CMD (0x1912)
	// as a Write Without Response. Non-blocking on the BLE stack.
	WriteCommand(ctx context.Context, data []byte) error

	// SubscribeNotify subscribes to CHAR_NOTIFY (0x1911) and returns a
	// channel delivering raw 20-byte packets. The channel is closed when
	// ctx is cancelled.
	SubscribeNotify(ctx context.Context) (<-chan []byte, error)

	// GatewayMAC returns the MAC address of the connected node — required
	// to build command and notification nonces.
	GatewayMAC() MACAddress
}

// ScanFilter narrows a BLE scan for Pixie devices.
type ScanFilter struct {
	// MeshName matches against the BLE local name. Empty matches any mesh.
	MeshName string
	// NetworkID matches against the 4-byte network ID in advert bytes
	// 11-14. Zero matches any network.
	NetworkID uint32
	// GatewayOnly, if true, keeps only advertisements whose DeviceType
	// equals [DeviceTypeGateway].
	GatewayOnly bool
}
