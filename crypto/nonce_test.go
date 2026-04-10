package crypto

import "testing"

func TestCommandNonce(t *testing.T) {
	// MAC: AA:BB:CC:DD:EE:FF → gwMAC[5]=FF, [4]=EE, [3]=DD, [2]=CC
	gwMAC := [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	sno := [3]byte{0x01, 0x42, 0x43}

	nonce := CommandNonce(gwMAC, sno)

	expected := [8]byte{0xFF, 0xEE, 0xDD, 0xCC, 0x01, 0x01, 0x42, 0x43}
	if nonce != expected {
		t.Errorf("CommandNonce:\n  got:  %x\n  want: %x", nonce, expected)
	}
}

func TestNotificationNonce(t *testing.T) {
	// Notification nonce uses only 3 MAC bytes and includes srcAddr.
	gwMAC := [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	sno := [3]byte{0x05, 0x06, 0x07}
	srcAddr := uint16(0x0102) // little-endian: lo=0x02, hi=0x01

	nonce := NotificationNonce(gwMAC, sno, srcAddr)

	expected := [8]byte{0xFF, 0xEE, 0xDD, 0x05, 0x06, 0x07, 0x02, 0x01}
	if nonce != expected {
		t.Errorf("NotificationNonce:\n  got:  %x\n  want: %x", nonce, expected)
	}
}
