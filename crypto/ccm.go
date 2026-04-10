package crypto

// Encrypt encrypts a plaintext command payload using the Telink AES-CCM scheme.
//
// It returns the full wire packet: sno(3) || tag(2) || ciphertext(N).
//
// The sno (sequence number) is 3 bytes: [seqNum, saltLo, saltHi] where seqNum
// is a per-command counter and salt is a per-session random value.
func Encrypt(sk [16]byte, nonce [8]byte, sno [3]byte, plaintext []byte) []byte {
	tag := cbcMAC(sk, nonce, plaintext)
	ct := ctr(sk, nonce, plaintext)

	packet := make([]byte, 3+2+len(ct))
	copy(packet[0:3], sno[:])
	copy(packet[3:5], tag[:])
	copy(packet[5:], ct)

	return packet
}

// Decrypt decrypts and verifies a notification payload.
//
// It takes the tag and ciphertext extracted from the notification wire format
// and returns the decrypted plaintext. Returns ErrTagMismatch if the CBC-MAC
// tag does not match.
func Decrypt(sk [16]byte, nonce [8]byte, tag [2]byte, ciphertext []byte) ([]byte, error) {
	// CTR mode is its own inverse.
	plaintext := ctr(sk, nonce, ciphertext)

	// Verify the tag against the decrypted plaintext.
	expected := cbcMAC(sk, nonce, plaintext)
	if tag != expected {
		return nil, ErrTagMismatch
	}

	return plaintext, nil
}

// cbcMAC computes the 2-byte truncated CBC-MAC authentication tag.
//
//	B0         = nonce[8] || data_len(1) || 0x00*7    (16 bytes)
//	mac_state  = ReversedAES(sk, B0)
//
//	for i in 0..len(data)-1:
//	    mac_state[i & 0xf] ^= data[i]
//	    if (i & 0xf) == 0xf  OR  i == len(data)-1:
//	        mac_state = ReversedAES(sk, mac_state)
//
//	tag = mac_state[0:2]
func cbcMAC(sk [16]byte, nonce [8]byte, data []byte) [2]byte {
	var b0 [16]byte
	copy(b0[:8], nonce[:])
	b0[8] = byte(len(data))
	// bytes 9-15 are zero (from array init)

	state := ReversedAES(sk, b0)

	for i, d := range data {
		state[i&0xf] ^= d
		if (i&0xf) == 0xf || i == len(data)-1 {
			state = ReversedAES(sk, state)
		}
	}

	return [2]byte{state[0], state[1]}
}

// ctr performs CTR-mode encryption/decryption (symmetric operation).
//
//	ctr_block = 0x00 || nonce[8] || 0x00*7     (16 bytes)
//
//	for i in 0..len(data)-1:
//	    if (i & 0xf) == 0:
//	        keystream = ReversedAES(sk, ctr_block)
//	        ctr_block[0]++
//	    output[i] = data[i] XOR keystream[i & 0xf]
func ctr(sk [16]byte, nonce [8]byte, data []byte) []byte {
	var ctrBlock [16]byte
	copy(ctrBlock[1:9], nonce[:])
	// byte 0 and bytes 9-15 are zero

	out := make([]byte, len(data))
	var keystream [16]byte

	for i, d := range data {
		if (i & 0xf) == 0 {
			keystream = ReversedAES(sk, ctrBlock)
			ctrBlock[0]++
		}
		out[i] = d ^ keystream[i&0xf]
	}

	return out
}
