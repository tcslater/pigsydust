// Package command builds the plaintext payloads of SAL Pixie / Telink BLE
// mesh commands. Builders return [Command] values; [Command.Encode] serialises
// them to the bytes that a client encrypts and writes to CHAR_CMD.
//
// All wire details match pigsydust-py/docs/PROTOCOL-REFERENCE.md.
package command

import (
	"encoding/binary"

	"github.com/tcslater/pigsydust/protocol"
)

// Command is a single plaintext command payload: destination, opcode,
// vendor, data, and the target plaintext length (which determines the
// zero-padding applied by [Command.Encode]).
//
// Most opcodes use 15-byte plaintext; status queries use 10, polls 7. See
// the per-builder documentation.
type Command struct {
	// Destination mesh address (individual, group, or broadcast).
	Destination uint16
	// Opcode — the 6-bit opcode byte (not yet shifted into the op_type
	// field). [Command.Encode] applies the op_type shift.
	Opcode byte
	// Vendor ID (little-endian on the wire). Usually [protocol.VendorSkytone].
	Vendor uint16
	// Data is the opcode-specific payload.
	Data []byte
	// PlaintextLen is the total size of the encoded plaintext, including
	// the 5-byte header (dst + opcode + vendor) and trailing zero pad.
	PlaintextLen int
}

// Encode serialises the command to its plaintext byte representation:
//
//	dst(2 LE) || opcode(1) || vendor(2 LE) || data(N) || zero_pad
//
// The first byte of the opcode field is ((op_type << 6) | (op & 0x3F)) with
// op_type = 3 (client-originated command).
func (c Command) Encode() []byte {
	buf := make([]byte, c.PlaintextLen)
	binary.LittleEndian.PutUint16(buf[0:2], c.Destination)
	buf[2] = (protocol.OpTypeClient << 6) | (c.Opcode & 0x3F)
	binary.LittleEndian.PutUint16(buf[3:5], c.Vendor)
	copy(buf[5:], c.Data)
	return buf
}
