package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	sk := DeriveSessionKey("Smart Light", "12345678",
		[8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		[8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
	)

	gwMAC := [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	sno := [3]byte{0x01, 0x42, 0x43}
	nonce := CommandNonce(gwMAC, sno)

	// 15-byte plaintext (typical on/off command).
	plaintext := []byte{
		0xff, 0xff, // dst broadcast
		0xed,       // opcode
		0x69, 0x69, // vendor
		0x01,                                     // state=ON
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // padding
	}

	packet := Encrypt(sk, nonce, sno, plaintext)

	// Packet format: sno(3) || tag(2) || ciphertext(N)
	if len(packet) != 3+2+len(plaintext) {
		t.Fatalf("packet length: got %d, want %d", len(packet), 3+2+len(plaintext))
	}

	// Verify sno is at the front.
	if !bytes.Equal(packet[0:3], sno[:]) {
		t.Errorf("sno mismatch: got %x, want %x", packet[0:3], sno[:])
	}

	// Extract tag and ciphertext, then decrypt.
	var tag [2]byte
	copy(tag[:], packet[3:5])
	ciphertext := packet[5:]

	decrypted, err := Decrypt(sk, nonce, tag, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("round-trip mismatch:\n  got:  %x\n  want: %x", decrypted, plaintext)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	sk1 := [16]byte{1}
	sk2 := [16]byte{2}

	nonce := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	sno := [3]byte{0x00, 0x00, 0x00}
	plaintext := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	packet := Encrypt(sk1, nonce, sno, plaintext)

	var tag [2]byte
	copy(tag[:], packet[3:5])
	ciphertext := packet[5:]

	_, err := Decrypt(sk2, nonce, tag, ciphertext)
	if err != ErrTagMismatch {
		t.Errorf("expected ErrTagMismatch, got %v", err)
	}
}

func TestEncrypt_DifferentSNO(t *testing.T) {
	sk := [16]byte{0x42}
	nonce1 := [8]byte{1, 2, 3, 4, 5, 0, 0, 0}
	nonce2 := [8]byte{1, 2, 3, 4, 5, 1, 0, 0}
	sno1 := [3]byte{0}
	sno2 := [3]byte{1}
	plaintext := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	p1 := Encrypt(sk, nonce1, sno1, plaintext)
	p2 := Encrypt(sk, nonce2, sno2, plaintext)

	// Different SNOs should produce different ciphertexts.
	if bytes.Equal(p1[5:], p2[5:]) {
		t.Error("different SNOs should produce different ciphertexts")
	}
}

func TestEncryptDecrypt_VariousLengths(t *testing.T) {
	sk := [16]byte{0xAA, 0xBB, 0xCC}
	nonce := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	sno := [3]byte{0x01, 0x02, 0x03}

	for _, length := range []int{7, 10, 15} {
		plaintext := make([]byte, length)
		for i := range plaintext {
			plaintext[i] = byte(i + 1)
		}

		packet := Encrypt(sk, nonce, sno, plaintext)

		var tag [2]byte
		copy(tag[:], packet[3:5])
		ciphertext := packet[5:]

		decrypted, err := Decrypt(sk, nonce, tag, ciphertext)
		if err != nil {
			t.Fatalf("length %d: Decrypt failed: %v", length, err)
		}

		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("length %d: round-trip mismatch", length)
		}
	}
}
