//go:build !linux

package ble

import (
	"context"
	"fmt"

	"tinygo.org/x/bluetooth"
)

// tinygoCharIO implements charIO over tinygo's DeviceCharacteristic.
type tinygoCharIO struct {
	ch bluetooth.DeviceCharacteristic
}

func (c *tinygoCharIO) writeRequest(_ context.Context, data []byte) error {
	// On Darwin tinygo's Write issues an ATT Write Request (with response).
	if _, err := c.ch.Write(data); err != nil {
		return fmt.Errorf("ble: write: %w", err)
	}
	return nil
}

func (c *tinygoCharIO) writeCommand(_ context.Context, data []byte) error {
	if _, err := c.ch.WriteWithoutResponse(data); err != nil {
		return fmt.Errorf("ble: write without response: %w", err)
	}
	return nil
}

func (c *tinygoCharIO) read(_ context.Context) ([]byte, error) {
	buf := make([]byte, 64)
	n, err := c.ch.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (c *tinygoCharIO) subscribe(ctx context.Context) (<-chan []byte, error) {
	ch := make(chan []byte, 64)

	err := c.ch.EnableNotifications(func(buf []byte) {
		packet := make([]byte, len(buf))
		copy(packet, buf)
		select {
		case ch <- packet:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("ble: enable notifications: %w", err)
	}

	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return ch, nil
}
