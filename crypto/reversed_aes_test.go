package crypto

import (
	"crypto/aes"
	"testing"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

func TestReversedAES_RoundTrip(t *testing.T) {
	// Verify that ReversedAES is consistent: applying it with known
	// inputs produces a deterministic result, and the reversal logic
	// matches manual AES with reversed inputs.
	key := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	plaintext := [16]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00}

	result := ReversedAES(key, plaintext)

	// Manually compute the same thing.
	rk := byteutil.Reverse16(key)
	rp := byteutil.Reverse16(plaintext)
	block, err := aes.NewCipher(rk[:])
	if err != nil {
		t.Fatal(err)
	}
	var ct [16]byte
	block.Encrypt(ct[:], rp[:])
	expected := byteutil.Reverse16(ct)

	if result != expected {
		t.Errorf("ReversedAES mismatch:\n  got:  %x\n  want: %x", result, expected)
	}
}

func TestReversedAES_DifferentInputs(t *testing.T) {
	// Two different keys should produce different results.
	key1 := [16]byte{1}
	key2 := [16]byte{2}
	pt := [16]byte{0xff}

	r1 := ReversedAES(key1, pt)
	r2 := ReversedAES(key2, pt)

	if r1 == r2 {
		t.Error("different keys produced the same result")
	}
}

func TestReversedAES_ZeroKey(t *testing.T) {
	// All-zero key and plaintext should still produce a valid result.
	var key, pt [16]byte
	result := ReversedAES(key, pt)

	// Just verify it doesn't panic and produces non-trivial output.
	var zero [16]byte
	if result == zero {
		t.Error("all-zero inputs should still produce non-zero AES output")
	}
}
