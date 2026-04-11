package command

import (
	"time"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

// SetUTC builds a time synchronisation broadcast (opcode 0xc5).
//
// This must be sent on every connection — mesh devices have no persistent
// RTC. The timezone byte is always 0x00 because the firmware offsets its
// internal clock by this value, and alarm records store times in UTC.
//
// The mesh responds with a burst of 0xdc status notifications from every device.
func SetUTC(now time.Time) Command {
	data := make([]byte, 5)
	byteutil.PutLE32(data[0:4], uint32(now.Unix()))
	data[4] = 0x00 // timezone byte: MUST be zero

	return Command{
		Destination:  AddrBroadcast,
		Opcode:       0xc5,
		Vendor:       VendorSkytone,
		Data:         data,
		PlaintextLen: 15,
	}
}
