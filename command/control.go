package command

import "github.com/tcslater/pigsydust/protocol"

// OnOff builds a turn-on / turn-off command (opcode 0xED, 15-byte plaintext).
// Works for individual device, group, and broadcast destinations.
func OnOff(dst uint16, on bool) Command {
	state := byte(0x00)
	if on {
		state = 0x01
	}
	return Command{
		Destination:  dst,
		Opcode:       protocol.OpOnOff,
		Vendor:       protocol.VendorSkytone,
		Data:         []byte{state},
		PlaintextLen: 15,
	}
}

// GroupOnOff builds the alternative group on/off command (opcode 0xE7,
// 15-byte plaintext). The group address is encoded both as dst and in the
// payload tail, with a fixed 0x10 byte between state and the trailing group
// address.
//
// Both 0xED and 0xE7 work for group addressing; 0xE7 is provided for parity
// with firmware captures that prefer it.
func GroupOnOff(group uint16, on bool) Command {
	state := byte(0x0D) // OFF
	if on {
		state = 0x0E
	}
	return Command{
		Destination: group,
		Opcode:      protocol.OpGroupOnOff,
		Vendor:      protocol.VendorSkytone,
		Data: []byte{
			state, 0x00, 0x10, 0x00, 0x00, 0x00,
			byte(group), byte(group >> 8),
		},
		PlaintextLen: 15,
	}
}
