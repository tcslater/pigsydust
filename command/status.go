package command

import (
	"encoding/binary"
	"time"

	"github.com/tcslater/pigsydust/protocol"
)

// StatusQuery builds the broadcast status query (opcode 0xC5, 10-byte
// plaintext). All devices respond with a 0xDC broadcast status notification.
//
// Bytes [0-1] of data are session-variable (firmware ignores them);
// bytes [2-4] carry the fixed tag 0xd7 0x69 0x00.
func StatusQuery() Command {
	return Command{
		Destination:  uint16(protocol.OpTypeClient), // placeholder replaced below
		Opcode:       protocol.OpStatusQuery,
		Vendor:       protocol.VendorSkytone,
		Data:         []byte{0x00, 0x00, 0xD7, 0x69, 0x00},
		PlaintextLen: 10,
	}.withDst(0xFFFF)
}

// StatusPoll builds a status poll (opcode 0xDA, 7-byte plaintext). Vendor
// is [protocol.VendorSkytoneAlt] (0x0211) rather than the usual
// [protocol.VendorSkytone].
//
// Use dst = [pigsydust.AddrBroadcastPoll] (0x7FFF) for a broadcast-poll
// keepalive or a specific device address to poll that device. This opcode
// is also the wake-up prerequisite before an LED query.
func StatusPoll(dst uint16) Command {
	return Command{
		Destination:  dst,
		Opcode:       protocol.OpStatusPoll,
		Vendor:       protocol.VendorSkytoneAlt,
		Data:         []byte{0x10, 0x00},
		PlaintextLen: 7,
	}
}

// SetUTC builds the broadcast time-sync command (opcode 0xC5, 15-byte
// plaintext).
//
//	tv_sec(4 LE) || tz(1)
//
// The tz byte must be 0x00 — the firmware offsets its internal clock by
// this value, and since alarm records store UTC any non-zero value causes
// schedule misfires.
//
// This must be sent on every connection; mesh devices have no persistent
// RTC and rely on this broadcast to anchor their schedule clock. The mesh
// responds with a burst of 0xDC status notifications from every device.
func SetUTC(now time.Time) Command {
	secs := uint32(now.Unix())
	data := make([]byte, 5)
	binary.LittleEndian.PutUint32(data[:4], secs)
	// data[4] = 0x00 (tz byte — must be zero).
	return Command{
		Destination:  0xFFFF,
		Opcode:       protocol.OpStatusQuery,
		Vendor:       protocol.VendorSkytone,
		Data:         data,
		PlaintextLen: 15,
	}
}

// withDst returns a copy of c with Destination set to dst.
func (c Command) withDst(dst uint16) Command {
	c.Destination = dst
	return c
}
