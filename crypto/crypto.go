// Package crypto implements the cryptographic primitives of the SAL Pixie
// / Telink BLE mesh protocol.
//
// All AES operations use a "reversed AES" convention where the key,
// plaintext, and ciphertext bytes are reversed before and after standard
// AES-128-ECB. This byte-reversal is applied at every level of the
// protocol — login handshake, session key derivation, and per-packet
// AES-CCM encryption.
//
// See ../protocol and the reference document at
// pigsydust-py/docs/PROTOCOL-REFERENCE.md for wire-format details.
package crypto

import (
	"crypto/aes"
	"fmt"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

// ReversedAES applies AES-128-ECB with key, plaintext, and ciphertext bytes
// reversed — the "Telink convention" used at every layer of the mesh
// protocol.
//
//	1. Reverse 16 bytes of key.
//	2. Reverse 16 bytes of plaintext.
//	3. AES-128-ECB encrypt with reversed inputs.
//	4. Reverse 16 bytes of ciphertext.
func ReversedAES(key, plaintext [16]byte) [16]byte {
	rk := byteutil.Reverse16(key)
	rp := byteutil.Reverse16(plaintext)

	c, err := aes.NewCipher(rk[:])
	if err != nil {
		// aes.NewCipher only errors on bad key length; 16 is always valid.
		panic(fmt.Sprintf("pigsydust/crypto: aes.NewCipher: %v", err))
	}

	var ct [16]byte
	c.Encrypt(ct[:], rp[:])
	return byteutil.Reverse16(ct)
}

// CommandNonce builds the 8-byte nonce used for encrypting commands.
//
//	gwMAC[5] || gwMAC[4] || gwMAC[3] || gwMAC[2] || 0x01 || sno[0..3]
//
// gwMAC is in standard printed order (index 0 = AA, index 5 = FF).
func CommandNonce(gwMAC [6]byte, sno [3]byte) [8]byte {
	return [8]byte{
		gwMAC[5], gwMAC[4], gwMAC[3], gwMAC[2],
		0x01,
		sno[0], sno[1], sno[2],
	}
}

// NotificationNonce builds the 8-byte nonce used for decrypting
// notifications.
//
//	gwMAC[5] || gwMAC[4] || gwMAC[3] || sno[0..3] || srcLo || srcHi
//
// Note: only 3 MAC bytes (not 4), and srcAddr replaces the 0x01 constant
// used in the command nonce. Getting this wrong is the most common reason
// a Telink implementation silently fails to decrypt notifications.
func NotificationNonce(gwMAC [6]byte, sno [3]byte, srcAddr uint16) [8]byte {
	return [8]byte{
		gwMAC[5], gwMAC[4], gwMAC[3],
		sno[0], sno[1], sno[2],
		byte(srcAddr), byte(srcAddr >> 8),
	}
}

// cbcMAC computes the 2-byte truncated CBC-MAC authentication tag.
func cbcMAC(sk [16]byte, nonce [8]byte, data []byte) [2]byte {
	var b0 [16]byte
	copy(b0[:8], nonce[:])
	b0[8] = byte(len(data))
	// b0[9..15] remain 0.

	state := ReversedAES(sk, b0)

	for i, d := range data {
		state[i&0xF] ^= d
		if (i&0xF) == 0xF || i == len(data)-1 {
			state = ReversedAES(sk, state)
		}
	}

	return [2]byte{state[0], state[1]}
}

// ctrCrypt applies CTR-mode encryption (or decryption — it's symmetric).
func ctrCrypt(sk [16]byte, nonce [8]byte, data []byte) []byte {
	var ctrBlock [16]byte
	copy(ctrBlock[1:9], nonce[:])
	// ctrBlock[0] is the counter, starts at 0.

	out := make([]byte, len(data))
	var keystream [16]byte

	for i, d := range data {
		if (i & 0xF) == 0 {
			keystream = ReversedAES(sk, ctrBlock)
			ctrBlock[0]++
		}
		out[i] = d ^ keystream[i&0xF]
	}
	return out
}

// Encrypt encrypts a plaintext command payload into the on-wire packet:
//
//	sno(3) || tag(2) || ciphertext(N)
//
// sno is the 3-byte sequence number that will be written into the packet
// header (also used for nonce construction — callers must build the nonce
// from the same sno via [CommandNonce]).
func Encrypt(sk [16]byte, nonce [8]byte, sno [3]byte, plaintext []byte) []byte {
	tag := cbcMAC(sk, nonce, plaintext)
	ct := ctrCrypt(sk, nonce, plaintext)

	packet := make([]byte, 0, 3+2+len(ct))
	packet = append(packet, sno[:]...)
	packet = append(packet, tag[:]...)
	packet = append(packet, ct...)
	return packet
}

// Decrypt verifies and decrypts a ciphertext using the expected 2-byte tag.
// Returns [pigsydust.ErrTagMismatch] (as a wrapped error message) if the
// CBC-MAC verification fails.
func Decrypt(sk [16]byte, nonce [8]byte, tag [2]byte, ciphertext []byte) ([]byte, error) {
	plaintext := ctrCrypt(sk, nonce, ciphertext)
	expected := cbcMAC(sk, nonce, plaintext)
	if expected != tag {
		return nil, fmt.Errorf("pigsydust/crypto: CBC-MAC tag mismatch")
	}
	return plaintext, nil
}
