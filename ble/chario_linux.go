//go:build linux

package ble

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
)

// linuxCharIO implements charIO over BlueZ GattCharacteristic1 via godbus.
//
// Writes go through BlueZ's WriteValue with an explicit type option
// (request or command). Notifications bypass BlueZ entirely — see
// subscribe below for the why.
type linuxCharIO struct {
	bus  *dbus.Conn
	path dbus.ObjectPath
}

func newLinuxCharIO(bus *dbus.Conn, path dbus.ObjectPath) *linuxCharIO {
	return &linuxCharIO{bus: bus, path: path}
}

func (c *linuxCharIO) writeRequest(ctx context.Context, data []byte) error {
	return c.writeValue(ctx, data, map[string]dbus.Variant{
		"type": dbus.MakeVariant("request"),
	})
}

func (c *linuxCharIO) writeCommand(ctx context.Context, data []byte) error {
	return c.writeValue(ctx, data, map[string]dbus.Variant{
		"type": dbus.MakeVariant("command"),
	})
}

func (c *linuxCharIO) writeValue(ctx context.Context, data []byte, opts map[string]dbus.Variant) error {
	obj := c.bus.Object(bluezBusName, c.path)
	call := obj.CallWithContext(ctx, bluezGattCharIface+".WriteValue", 0, data, opts)
	if call.Err != nil {
		return fmt.Errorf("ble: WriteValue %s: %w", c.path, call.Err)
	}
	return nil
}

func (c *linuxCharIO) read(ctx context.Context) ([]byte, error) {
	obj := c.bus.Object(bluezBusName, c.path)
	opts := map[string]dbus.Variant{}
	var out []byte
	if err := obj.CallWithContext(ctx, bluezGattCharIface+".ReadValue", 0, opts).Store(&out); err != nil {
		return nil, fmt.Errorf("ble: ReadValue %s: %w", c.path, err)
	}
	return out, nil
}

// subscribe arms notifications on this characteristic. Matches pigsydust-py's
// Linux strategy: *do not* call BlueZ's StartNotify. Telink's firmware
// exposes a CCCD descriptor that silently drops ATT writes to it, so
// BlueZ's StartNotify either hangs or disconnects us (we've observed the
// latter — ATT 0x0e followed by immediate connection loss).
//
// Instead we:
//  1. Look up the characteristic's value-attribute handle via BlueZ's Handle
//     property (that's the declaration handle; value handle is Handle+1).
//  2. Open a raw HCI socket (AF_BLUETOOTH/BTPROTO_HCI) and parse ACL frames
//     for ATT HANDLE_VALUE_NTF (0x1B) PDUs on our handle.
//  3. Write 0x01 to the characteristic's value via ATT Write Request —
//     that's what actually arms Telink's firmware notify pump.
//
// Requires CAP_NET_RAW on the binary (or running as root). Without it, the
// raw HCI socket fails at open and subscribe returns the error.
func (c *linuxCharIO) subscribe(ctx context.Context) (<-chan []byte, error) {
	declHandle, err := c.readHandle()
	if err != nil {
		return nil, err
	}
	valueHandle := declHandle + 1

	fd, err := openHCIRaw()
	if err != nil {
		return nil, fmt.Errorf("ble: opening raw HCI socket: %w", err)
	}

	out := make(chan []byte, 64)
	go readHCINotifications(ctx, fd, valueHandle, out)

	return out, nil
}

// readHandle fetches BlueZ's Handle property for this characteristic.
func (c *linuxCharIO) readHandle() (uint16, error) {
	v, err := getProperty(c.bus, c.path, bluezGattCharIface, "Handle")
	if err != nil {
		return 0, fmt.Errorf("ble: reading char Handle: %w", err)
	}
	h, ok := v.Value().(uint16)
	if !ok {
		return 0, fmt.Errorf("ble: Handle property unexpected type %T", v.Value())
	}
	return h, nil
}

// openHCIRaw opens a raw HCI socket bound to hci0 with a filter that passes
// only ACL data packets (type 2). Returns an fd ready to Read from.
func openHCIRaw() (int, error) {
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_RAW, unix.BTPROTO_HCI)
	if err != nil {
		return -1, err
	}
	// struct hci_filter { type_mask uint32; event_mask[2] uint32; opcode uint16; pad uint16 }
	// Size = 4 + 8 + 2 + 2 = 16 bytes.
	var filter [16]byte
	// HCI_ACL_DATA_PKT = 2.
	binary.LittleEndian.PutUint32(filter[0:4], 1<<2)
	// SOL_HCI = 0, HCI_FILTER = 2.
	if err := unix.SetsockoptString(fd, 0, 2, string(filter[:])); err != nil {
		unix.Close(fd)
		return -1, fmt.Errorf("setting HCI filter: %w", err)
	}
	sa := &unix.SockaddrHCI{Dev: 0, Channel: unix.HCI_CHANNEL_RAW}
	if err := unix.Bind(fd, sa); err != nil {
		unix.Close(fd)
		return -1, fmt.Errorf("binding HCI socket: %w", err)
	}
	return fd, nil
}

// readHCINotifications reads ACL packets from fd and forwards the bodies of
// ATT HANDLE_VALUE_NTF PDUs matching valueHandle to out. Exits when ctx
// is cancelled; closes out on exit.
func readHCINotifications(ctx context.Context, fd int, valueHandle uint16, out chan<- []byte) {
	defer close(out)
	defer unix.Close(fd)

	// Make the socket non-blocking so we can poll + honour ctx.
	if err := unix.SetNonblock(fd, true); err != nil {
		log.Printf("ble: SetNonblock hci: %v", err)
		return
	}

	buf := make([]byte, 1024)
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		// Use poll(2) with a short timeout so we wake up for ctx cancellation.
		pfd := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		if _, err := unix.Poll(pfd, 200); err != nil {
			if err == unix.EINTR {
				continue
			}
			log.Printf("ble: poll hci: %v", err)
			return
		}
		if pfd[0].Revents&unix.POLLIN == 0 {
			continue
		}

		n, err := unix.Read(fd, buf)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			return
		}
		if n < 12 {
			continue
		}
		// HCI packet framing: type(1) + ACL header.
		// ACL: handle+flags(2) total_len(2) l2cap_len(2) l2cap_cid(2) att...
		if buf[0] != 2 { // HCI_ACL_DATA_PKT
			continue
		}
		l2capCID := binary.LittleEndian.Uint16(buf[7:9])
		if l2capCID != 0x0004 { // ATT_CID
			continue
		}
		if n < 12 {
			continue
		}
		attOpcode := buf[9]
		if attOpcode != 0x1B { // HANDLE_VALUE_NTF
			continue
		}
		attHandle := binary.LittleEndian.Uint16(buf[10:12])
		if attHandle != valueHandle {
			continue
		}
		packet := make([]byte, n-12)
		copy(packet, buf[12:n])
		select {
		case out <- packet:
		case <-ctx.Done():
			return
		default:
			// drop
		}
	}
}
