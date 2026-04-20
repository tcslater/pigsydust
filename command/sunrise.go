package command

import "github.com/tcslater/pigsydust/protocol"

// SunriseSunset builds the three-fragment sunrise/sunset push (opcode
// 0xD0, 15-byte plaintext each).
//
//	frag_index(1) || compressed_sun_data[8]
//
// The third fragment's data trails with an XOR checksum byte over the
// 24 payload bytes. The payload carries day-of-epoch and compressed
// astronomical data — format not yet fully documented.
//
// fragments[0..2] must be sent in order, and each must be 8 bytes of
// compressed data. The caller is responsible for building the compressed
// data; this builder just emits the three outer frames.
func SunriseSunset(dst uint16, fragments [3][8]byte) [3]Command {
	var out [3]Command
	var xor byte
	for _, f := range fragments {
		for _, b := range f {
			xor ^= b
		}
	}
	for i, f := range fragments {
		data := make([]byte, 1+len(f))
		data[0] = byte(i)
		copy(data[1:], f[:])
		// Last fragment carries the xor checksum at the end. Plaintext
		// length 15 leaves room: 5 header + 1 index + 8 payload = 14; the
		// final zero-pad byte is replaced by xor for i == 2.
		out[i] = Command{
			Destination:  dst,
			Opcode:       protocol.OpSunriseSunset,
			Vendor:       protocol.VendorSkytone,
			Data:         data,
			PlaintextLen: 15,
		}
	}
	// Append the XOR checksum as the last byte of the third fragment's
	// data; Encode will still zero-pad to PlaintextLen.
	out[2].Data = append(out[2].Data, xor)
	return out
}
