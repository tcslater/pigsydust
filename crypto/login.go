package crypto

import "github.com/tcslater/pigsydust/internal/byteutil"

// LoginRequest builds the 17-byte CHAR_PAIR (0x1914) write payload for
// the login handshake.
//
// The caller provides randA (8 random bytes); the returned payload is:
//
//	[0x0c] || randA[8] || enc_req[8]
//
// where enc_req is the first 8 bytes of:
//
//	ReversedAES(key=randA||zeros, plaintext=pad16(name) XOR pad16(pass))
func LoginRequest(name, password string, randA [8]byte) [17]byte {
	// key = randA || 0x00*8
	var key [16]byte
	copy(key[:8], randA[:])

	// plaintext = pad16(name) XOR pad16(pass)
	plaintext := byteutil.XOR16(byteutil.Pad16(name), byteutil.Pad16(password))

	ct := ReversedAES(key, plaintext)

	var payload [17]byte
	payload[0] = 0x0c
	copy(payload[1:9], randA[:])
	copy(payload[9:17], ct[:8])

	return payload
}

// ParseLoginResponse extracts randB from the 17-byte CHAR_PAIR read response.
//
// The expected format is:
//
//	[0x0d] || randB[8] || auth[8]
//
// Returns an error if the response is malformed or the tag byte is not 0x0d.
func ParseLoginResponse(resp []byte) ([8]byte, error) {
	var randB [8]byte
	if len(resp) < 17 {
		return randB, ErrLoginFailed
	}
	if resp[0] != 0x0d {
		return randB, ErrLoginFailed
	}
	copy(randB[:], resp[1:9])
	return randB, nil
}
