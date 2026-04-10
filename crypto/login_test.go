package crypto

import "testing"

func TestLoginRequest_Format(t *testing.T) {
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	payload := LoginRequest("Smart Light", "12345678", randA)

	// Tag byte must be 0x0c.
	if payload[0] != 0x0c {
		t.Errorf("tag byte: got 0x%02x, want 0x0c", payload[0])
	}

	// Bytes 1-8 must be randA.
	for i := range 8 {
		if payload[1+i] != randA[i] {
			t.Errorf("randA[%d]: got 0x%02x, want 0x%02x", i, payload[1+i], randA[i])
		}
	}

	// Bytes 9-16 (enc_req) should be non-zero for non-trivial inputs.
	allZero := true
	for i := 9; i < 17; i++ {
		if payload[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("enc_req should not be all zeros")
	}
}

func TestParseLoginResponse_Valid(t *testing.T) {
	resp := make([]byte, 17)
	resp[0] = 0x0d
	for i := 1; i < 9; i++ {
		resp[i] = byte(i * 0x11)
	}

	randB, err := ParseLoginResponse(resp)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 8 {
		if randB[i] != byte((i+1)*0x11) {
			t.Errorf("randB[%d]: got 0x%02x, want 0x%02x", i, randB[i], byte((i+1)*0x11))
		}
	}
}

func TestParseLoginResponse_WrongTag(t *testing.T) {
	resp := make([]byte, 17)
	resp[0] = 0x0c // wrong tag

	_, err := ParseLoginResponse(resp)
	if err == nil {
		t.Error("expected error for wrong tag byte")
	}
}

func TestParseLoginResponse_TooShort(t *testing.T) {
	resp := make([]byte, 10) // too short
	resp[0] = 0x0d

	_, err := ParseLoginResponse(resp)
	if err == nil {
		t.Error("expected error for short response")
	}
}
