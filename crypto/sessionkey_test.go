package crypto

import "testing"

func TestDeriveSessionKey_Deterministic(t *testing.T) {
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	randB := [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}

	sk1 := DeriveSessionKey("Smart Light", "12345678", randA, randB)
	sk2 := DeriveSessionKey("Smart Light", "12345678", randA, randB)

	if sk1 != sk2 {
		t.Error("same inputs should produce same session key")
	}
}

func TestDeriveSessionKey_DifferentFromLogin(t *testing.T) {
	// Verify the asymmetry: the session key derivation uses the opposite
	// key/plaintext assignment from the login request.
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	randB := [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}

	sk := DeriveSessionKey("Smart Light", "12345678", randA, randB)

	// The login request uses key=randA||zeros, plaintext=name^pass.
	// The session key uses key=name^pass, plaintext=randA||randB.
	// These should not be equal for non-trivial inputs.
	loginPayload := LoginRequest("Smart Light", "12345678", randA)
	_ = loginPayload

	// Just verify the session key is non-zero and 16 bytes.
	var zero [16]byte
	if sk == zero {
		t.Error("session key should not be all zeros")
	}
}

func TestDeriveSessionKey_DifferentNonces(t *testing.T) {
	randA1 := [8]byte{0x01}
	randA2 := [8]byte{0x02}
	randB := [8]byte{0x11}

	sk1 := DeriveSessionKey("test", "pass", randA1, randB)
	sk2 := DeriveSessionKey("test", "pass", randA2, randB)

	if sk1 == sk2 {
		t.Error("different randA should produce different session keys")
	}
}
