// Package command provides builders for Telink BLE mesh protocol commands.
//
// Each builder function returns a [Command] struct that can be serialized
// to its plaintext byte representation via [Command.Encode]. The resulting
// plaintext is then encrypted by the crypto package before being written
// to the mesh.
//
// Addresses are represented as uint16 to avoid import cycles with the
// root package. Use the address constants defined in this package or
// cast from [pigsydust.Address].
package command

import "github.com/tcslater/pigsydust/internal/byteutil"

const (
	// VendorSkytone is the standard vendor ID for most commands.
	VendorSkytone uint16 = 0x6969

	// VendorSkytoneAlt is the alternate vendor ID used by status polls
	// and group queries.
	VendorSkytoneAlt uint16 = 0x0211

	// VendorLEDQuery is the unique vendor ID used by LED indicator queries.
	VendorLEDQuery uint16 = 0x696b

	// OpTypeClient is the operation type for client-to-device commands.
	// Wire opcode byte = (OpTypeClient << 6) | (opcode6 & 0x3f).
	OpTypeClient byte = 3

	// Well-known mesh addresses.
	AddrBroadcast            uint16 = 0xFFFF
	AddrBroadcastPoll        uint16 = 0x7FFF
	AddrScheduleCoordinator  uint16 = 0x0030
)

// Command represents a mesh protocol command before encryption.
type Command struct {
	Destination  uint16
	Opcode       byte
	Vendor       uint16
	Data         []byte
	PlaintextLen int // total plaintext length (7, 10, or 15)
}

// Encode serializes the command to its plaintext byte representation.
//
// The format is:
//
//	dst(2 LE) || opcode(1) || vendor(2 LE) || data(N) || zero_pad
func (c Command) Encode() []byte {
	buf := make([]byte, c.PlaintextLen)

	byteutil.PutLE16(buf[0:2], c.Destination)
	buf[2] = (OpTypeClient << 6) | (c.Opcode & 0x3f)
	byteutil.PutLE16(buf[3:5], c.Vendor)

	if len(c.Data) > 0 {
		copy(buf[5:], c.Data)
	}

	return buf
}
