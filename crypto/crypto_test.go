package crypto

import (
	"bytes"
	"crypto/aes"
	"errors"
	"testing"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

// --- ReversedAES ---

func TestReversedAESRoundTrip(t *testing.T) {
	key := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}
	pt := [16]byte{
		0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00,
	}

	got := ReversedAES(key, pt)

	// Manual computation against stdlib AES.
	rk := byteutil.Reverse16(key)
	rp := byteutil.Reverse16(pt)
	c, err := aes.NewCipher(rk[:])
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	var ct [16]byte
	c.Encrypt(ct[:], rp[:])
	want := byteutil.Reverse16(ct)

	if got != want {
		t.Errorf("ReversedAES mismatch\n got %x\nwant %x", got, want)
	}
}

func TestReversedAESDifferentKeys(t *testing.T) {
	var key1, key2, pt [16]byte
	key1[0] = 0x01
	key2[0] = 0x02
	pt[0] = 0xFF

	if ReversedAES(key1, pt) == ReversedAES(key2, pt) {
		t.Error("ReversedAES produced identical output for different keys")
	}
}

func TestReversedAESZeroKey(t *testing.T) {
	var key, pt [16]byte
	got := ReversedAES(key, pt)
	var zero [16]byte
	if got == zero {
		t.Error("ReversedAES(0, 0) returned zero — AES(0, 0) is not zero")
	}
}

// --- Login ---

func TestBuildLoginRequestFormat(t *testing.T) {
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	req := BuildLoginRequest("Smart Light", "12345678", randA)

	if req[0] != LoginRequestTag {
		t.Errorf("tag byte = 0x%02x, want 0x0c", req[0])
	}
	if !bytes.Equal(req[1:9], randA[:]) {
		t.Errorf("randA not echoed: %x", req[1:9])
	}
	var zero [8]byte
	if bytes.Equal(req[9:17], zero[:]) {
		t.Error("encReq is zero — encryption probably didn't run")
	}
}

func TestParseLoginResponseValid(t *testing.T) {
	var resp [17]byte
	resp[0] = LoginResponseTag
	for i := range 8 {
		resp[1+i] = byte((i + 1) * 0x11)
	}
	randB, err := ParseLoginResponse(resp[:])
	if err != nil {
		t.Fatalf("ParseLoginResponse: %v", err)
	}
	for i := range 8 {
		if randB[i] != byte((i+1)*0x11) {
			t.Errorf("randB[%d] = 0x%02x, want 0x%02x", i, randB[i], (i+1)*0x11)
		}
	}
}

func TestParseLoginResponseWrongTag(t *testing.T) {
	resp := make([]byte, 17)
	resp[0] = 0x0C // wrong — this is the request tag
	if _, err := ParseLoginResponse(resp); err == nil {
		t.Error("ParseLoginResponse with wrong tag returned nil error")
	}
}

func TestParseLoginResponseTooShort(t *testing.T) {
	resp := make([]byte, 10)
	resp[0] = LoginResponseTag
	if _, err := ParseLoginResponse(resp); err == nil {
		t.Error("ParseLoginResponse with short buffer returned nil error")
	}
}

// --- Session key ---

func TestDeriveSessionKeyDeterministic(t *testing.T) {
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	randB := [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	sk1 := DeriveSessionKey("Smart Light", "12345678", randA, randB)
	sk2 := DeriveSessionKey("Smart Light", "12345678", randA, randB)
	if sk1 != sk2 {
		t.Error("DeriveSessionKey is not deterministic")
	}
}

func TestDeriveSessionKeyNonzero(t *testing.T) {
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	randB := [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	sk := DeriveSessionKey("Smart Light", "12345678", randA, randB)
	var zero [16]byte
	if sk == zero {
		t.Error("derived session key is zero")
	}
}

func TestDeriveSessionKeyDifferentNonces(t *testing.T) {
	randA1 := [8]byte{0x01}
	randA2 := [8]byte{0x02}
	randB := [8]byte{0x11}
	sk1 := DeriveSessionKey("test", "pass", randA1, randB)
	sk2 := DeriveSessionKey("test", "pass", randA2, randB)
	if sk1 == sk2 {
		t.Error("DeriveSessionKey produced identical output for different randA")
	}
}

// --- Nonce construction ---

func TestCommandNonce(t *testing.T) {
	gwMAC := [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	sno := [3]byte{0x01, 0x42, 0x43}
	want := [8]byte{0xFF, 0xEE, 0xDD, 0xCC, 0x01, 0x01, 0x42, 0x43}
	if got := CommandNonce(gwMAC, sno); got != want {
		t.Errorf("CommandNonce = %x, want %x", got, want)
	}
}

func TestNotificationNonce(t *testing.T) {
	gwMAC := [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	sno := [3]byte{0x05, 0x06, 0x07}
	want := [8]byte{0xFF, 0xEE, 0xDD, 0x05, 0x06, 0x07, 0x02, 0x01}
	if got := NotificationNonce(gwMAC, sno, 0x0102); got != want {
		t.Errorf("NotificationNonce = %x, want %x", got, want)
	}
}

// --- Encrypt / Decrypt round-trip ---

func TestEncryptDecryptRoundTrip(t *testing.T) {
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	randB := [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	sk := DeriveSessionKey("Smart Light", "12345678", randA, randB)

	gwMAC := [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	sno := [3]byte{0x01, 0x42, 0x43}
	nonce := CommandNonce(gwMAC, sno)

	plaintext := []byte{
		0xFF, 0xFF, // dst = broadcast
		0xED,       // opcode
		0x69, 0x69, // vendor
		0x01, // state = ON
		0, 0, 0, 0, 0, 0, 0, 0, 0, // pad to 15
	}

	packet := Encrypt(sk, nonce, sno, plaintext)
	if len(packet) != 3+2+len(plaintext) {
		t.Fatalf("packet length = %d, want %d", len(packet), 3+2+len(plaintext))
	}
	if !bytes.Equal(packet[:3], sno[:]) {
		t.Errorf("sno header mismatch: %x vs %x", packet[:3], sno)
	}

	var tag [2]byte
	copy(tag[:], packet[3:5])
	got, err := Decrypt(sk, nonce, tag, packet[5:])
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("round-trip mismatch\n got %x\nwant %x", got, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	var sk1, sk2 [16]byte
	sk1[0] = 0x01
	sk2[0] = 0x02
	var nonce [8]byte
	for i := range nonce {
		nonce[i] = byte(i)
	}
	sno := [3]byte{}
	plaintext := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	packet := Encrypt(sk1, nonce, sno, plaintext)
	var tag [2]byte
	copy(tag[:], packet[3:5])

	if _, err := Decrypt(sk2, nonce, tag, packet[5:]); err == nil {
		t.Error("Decrypt with wrong key returned nil error")
	} else if !errors.Is(err, err) { // just exercise the path
		_ = err
	}
}

func TestEncryptDifferentSnoDiverges(t *testing.T) {
	var sk [16]byte
	sk[0] = 0x42
	nonce1 := [8]byte{1, 2, 3, 4, 5}
	nonce2 := [8]byte{1, 2, 3, 4, 5, 1}
	sno1 := [3]byte{}
	sno2 := [3]byte{0x01}
	plaintext := []byte{1, 2, 3, 4, 5}

	p1 := Encrypt(sk, nonce1, sno1, plaintext)
	p2 := Encrypt(sk, nonce2, sno2, plaintext)
	if bytes.Equal(p1[5:], p2[5:]) {
		t.Error("ciphertext identical for different nonces")
	}
}

func TestEncryptDecryptVariousLengths(t *testing.T) {
	sk := [16]byte{0xAA, 0xBB, 0xCC}
	var nonce [8]byte
	for i := range nonce {
		nonce[i] = byte(i + 1)
	}
	sno := [3]byte{0x01, 0x02, 0x03}

	for _, length := range []int{7, 10, 15} {
		pt := make([]byte, length)
		for i := range pt {
			pt[i] = byte(i + 1)
		}
		packet := Encrypt(sk, nonce, sno, pt)
		var tag [2]byte
		copy(tag[:], packet[3:5])
		got, err := Decrypt(sk, nonce, tag, packet[5:])
		if err != nil {
			t.Fatalf("length=%d: %v", length, err)
		}
		if !bytes.Equal(got, pt) {
			t.Errorf("length=%d: round-trip mismatch", length)
		}
	}
}
