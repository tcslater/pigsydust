package command

// StatusQuery builds a broadcast status query command (opcode 0xc5).
//
// Causes all mesh devices to respond with 0xdc status notifications.
// Uses a 10-byte plaintext.
func StatusQuery() Command {
	return Command{
		Destination:  AddrBroadcast,
		Opcode:       0xc5,
		Vendor:       VendorSkytone,
		Data:         []byte{0x00, 0x00, 0xd7, 0x69, 0x00},
		PlaintextLen: 10,
	}
}

// StatusPoll builds a status poll command (opcode 0xda).
//
// Elicits a 0xdb status notification from the target device.
// Use dst = 0x7FFF for broadcast poll.
// Uses a 7-byte plaintext with vendor 0x0211.
func StatusPoll(dst uint16) Command {
	return Command{
		Destination:  dst,
		Opcode:       0xda,
		Vendor:       VendorSkytoneAlt,
		Data:         []byte{0x10, 0x00},
		PlaintextLen: 7,
	}
}
