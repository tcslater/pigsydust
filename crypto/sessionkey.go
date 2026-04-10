package crypto

import "github.com/tcslater/piggsydust/internal/byteutil"

// DeriveSessionKey computes the per-session encryption key from the
// mesh credentials and the login nonces.
//
// Note the key/plaintext assignment is the opposite of the login request:
//
//	key       = pad16(name) XOR pad16(pass)
//	plaintext = randA[8] || randB[8]
//	sk        = ReversedAES(key, plaintext)
func DeriveSessionKey(name, password string, randA, randB [8]byte) [16]byte {
	key := byteutil.XOR16(byteutil.Pad16(name), byteutil.Pad16(password))

	var plaintext [16]byte
	copy(plaintext[:8], randA[:])
	copy(plaintext[8:], randB[:])

	return ReversedAES(key, plaintext)
}
