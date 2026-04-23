// Package byteutil provides low-level byte manipulation helpers
// for the Telink BLE mesh protocol.
package byteutil

import "encoding/binary"

// Pad16 pads or truncates s to exactly 16 bytes, zero-filling on the right.
func Pad16(s string) [16]byte {
	var b [16]byte
	copy(b[:], s)
	return b
}

// XOR16 returns the byte-wise XOR of two 16-byte arrays.
func XOR16(a, b [16]byte) [16]byte {
	var out [16]byte
	for i := range 16 {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// Reverse16 returns a copy of b with all 16 bytes reversed.
func Reverse16(b [16]byte) [16]byte {
	var out [16]byte
	for i := range 16 {
		out[i] = b[15-i]
	}
	return out
}

// PutLE16 writes v as a little-endian uint16 into dst[0:2].
func PutLE16(dst []byte, v uint16) {
	binary.LittleEndian.PutUint16(dst, v)
}

// LE16 reads a little-endian uint16 from b[0:2].
func LE16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

// PutLE32 writes v as a little-endian uint32 into dst[0:4].
func PutLE32(dst []byte, v uint32) {
	binary.LittleEndian.PutUint32(dst, v)
}

// LE32 reads a little-endian uint32 from b[0:4].
func LE32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

// XORFold returns the XOR of all bytes in b.
func XORFold(b []byte) byte {
	var x byte
	for _, v := range b {
		x ^= v
	}
	return x
}
