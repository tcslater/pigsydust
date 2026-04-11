package crypto

import (
	"crypto/aes"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

// ReversedAES performs the Telink reversed AES-128-ECB operation:
//
//	reverse all 16 bytes of key
//	reverse all 16 bytes of plaintext
//	ciphertext = AES-128-ECB(reversed_key, reversed_plaintext)
//	reverse all 16 bytes of ciphertext
//	return reversed_ciphertext
func ReversedAES(key, plaintext [16]byte) [16]byte {
	rk := byteutil.Reverse16(key)
	rp := byteutil.Reverse16(plaintext)

	block, err := aes.NewCipher(rk[:])
	if err != nil {
		// aes.NewCipher only fails for invalid key sizes; 16 is always valid.
		panic("pigsydust/crypto: " + err.Error())
	}

	var ct [16]byte
	block.Encrypt(ct[:], rp[:])

	return byteutil.Reverse16(ct)
}
