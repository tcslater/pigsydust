package crypto

// CommandNonce builds the 8-byte nonce for encrypting command packets.
//
// Layout:
//
//	gwMAC[5] || gwMAC[4] || gwMAC[3] || gwMAC[2] || 0x01 || sno[0] || sno[1] || sno[2]
//
// gwMAC is in standard printed order (AA:BB:CC:DD:EE:FF), so gwMAC[5]=FF, etc.
func CommandNonce(gwMAC [6]byte, sno [3]byte) [8]byte {
	return [8]byte{
		gwMAC[5], gwMAC[4], gwMAC[3], gwMAC[2],
		0x01,
		sno[0], sno[1], sno[2],
	}
}

// NotificationNonce builds the 8-byte nonce for decrypting notification packets.
//
// Layout:
//
//	gwMAC[5] || gwMAC[4] || gwMAC[3] || sno[0] || sno[1] || sno[2] || srcAddr_lo || srcAddr_hi
//
// Note: only 3 MAC bytes (not 4), and srcAddr replaces the constant 0x01 byte.
func NotificationNonce(gwMAC [6]byte, sno [3]byte, srcAddr uint16) [8]byte {
	return [8]byte{
		gwMAC[5], gwMAC[4], gwMAC[3],
		sno[0], sno[1], sno[2],
		byte(srcAddr), byte(srcAddr >> 8),
	}
}
