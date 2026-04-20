package ble

import (
	"context"
	"fmt"
	"sync"

	"github.com/tcslater/pigsydust"
)

// Transport implements [pigsydust.Transport] over a BLE connection
// established via [Adapter.Connect].
//
// All methods are safe for concurrent use.
type Transport struct {
	conn *Connection
	mu   sync.Mutex // serialises writes to avoid interleaving
}

// NewTransport creates a [pigsydust.Transport] from an established
// BLE connection.
func NewTransport(conn *Connection) *Transport {
	return &Transport{conn: conn}
}

// WritePair writes data to CHAR_PAIR (0x1914) using ATT Write Request
// (with response). Used for the login handshake.
func (t *Transport) WritePair(_ context.Context, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, err := t.conn.charPair.Write(data)
	if err != nil {
		return fmt.Errorf("ble: writing CHAR_PAIR: %w", err)
	}
	return nil
}

// ReadPair reads from CHAR_PAIR (0x1914). Used for the login response
// and heartbeat keepalive.
func (t *Transport) ReadPair(_ context.Context) ([]byte, error) {
	buf := make([]byte, 20)
	n, err := t.conn.charPair.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("ble: reading CHAR_PAIR: %w", err)
	}
	return buf[:n], nil
}

// WriteCommand writes an encrypted packet to CHAR_CMD (0x1912) using
// Write Without Response for best throughput.
func (t *Transport) WriteCommand(_ context.Context, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, err := t.conn.charCmd.WriteWithoutResponse(data)
	if err != nil {
		return fmt.Errorf("ble: writing CHAR_CMD: %w", err)
	}
	return nil
}

// SubscribeNotify subscribes to CHAR_NOTIFY (0x1911) and returns a channel
// that delivers raw 20-byte notification packets.
//
// Telink mesh firmware doesn't expose a standard CCCD on CHAR_NOTIFY — the
// iOS Pixie app enables notifications by writing 0x01 directly to the
// characteristic value. We still call EnableNotifications so the host BLE
// stack wires up the delivery callback and flips its internal "notifying"
// flag, then write 0x01 to actually kick the firmware.
//
// The channel is closed when the context is cancelled. Only one subscription
// can be active at a time.
func (t *Transport) SubscribeNotify(ctx context.Context) (<-chan []byte, error) {
	ch := make(chan []byte, 64)

	err := t.conn.charNotify.EnableNotifications(func(buf []byte) {
		packet := make([]byte, len(buf))
		copy(packet, buf)

		select {
		case ch <- packet:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("ble: enabling notifications: %w", err)
	}

	// ATT Write Request (with response). Write Without Response is silently
	// dropped by CoreBluetooth for this characteristic — the firmware only
	// arms its notify pump on a Write Request.
	if _, err := t.conn.charNotify.Write([]byte{0x01}); err != nil {
		return nil, fmt.Errorf("ble: kicking Telink notify: %w", err)
	}

	// Close the channel when context is done.
	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return ch, nil
}

// GatewayMAC returns the 6-byte MAC address of the connected gateway.
func (t *Transport) GatewayMAC() pigsydust.MACAddress {
	return t.conn.gwMAC
}
