package crypto

import (
	"fmt"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

// LoginRequestTag is the first byte of a pairing request packet (written
// to CHAR_PAIR), per the protocol.
const LoginRequestTag = 0x0C

// LoginResponseTag is the first byte of the pairing response packet
// (read back from CHAR_PAIR).
const LoginResponseTag = 0x0D

// BuildLoginRequest builds the 17-byte CHAR_PAIR login request.
//
//	0x0c || randA(8) || encReq(8)
//
// where:
//
//	key       = randA || 0x00*8        (16 bytes)
//	plaintext = pad16(name) XOR pad16(password)
//	encReq    = reversedAES(key, plaintext)[0:8]
//
// Callers must supply randA from a cryptographically secure source.
func BuildLoginRequest(name, password string, randA [8]byte) [17]byte {
	var key [16]byte
	copy(key[:8], randA[:])
	// key[8..15] are already zero.

	nameBuf := byteutil.Pad16(name)
	passBuf := byteutil.Pad16(password)
	pt := byteutil.XOR16(nameBuf, passBuf)

	ct := ReversedAES(key, pt)

	var req [17]byte
	req[0] = LoginRequestTag
	copy(req[1:9], randA[:])
	copy(req[9:17], ct[:8])
	return req
}

// ParseLoginResponse extracts randB from the 17-byte CHAR_PAIR login
// response. Returns an error if the response is malformed.
func ParseLoginResponse(resp []byte) ([8]byte, error) {
	var randB [8]byte
	if len(resp) < 17 {
		return randB, fmt.Errorf("pigsydust/crypto: login response too short (%d bytes)", len(resp))
	}
	if resp[0] != LoginResponseTag {
		return randB, fmt.Errorf("pigsydust/crypto: login response bad tag 0x%02x", resp[0])
	}
	copy(randB[:], resp[1:9])
	return randB, nil
}

// DeriveSessionKey derives the per-session 16-byte encryption key.
//
// Note: the key/plaintext assignment is the opposite of [BuildLoginRequest]:
//
//	key       = pad16(name) XOR pad16(password)
//	plaintext = randA || randB
//	sk        = reversedAES(key, plaintext)
//
// This asymmetry is the single most common implementation mistake — getting
// it wrong produces a valid-looking session that decrypts to garbage.
func DeriveSessionKey(name, password string, randA, randB [8]byte) [16]byte {
	key := byteutil.XOR16(byteutil.Pad16(name), byteutil.Pad16(password))

	var pt [16]byte
	copy(pt[:8], randA[:])
	copy(pt[8:16], randB[:])

	return ReversedAES(key, pt)
}
