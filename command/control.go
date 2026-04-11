package command

import "github.com/tcslater/pigsydust/internal/byteutil"

// OnOff builds an on/off command (opcode 0xed).
//
// Works for individual devices, groups, and broadcast (0xFFFF).
// State: true = ON, false = OFF.
func OnOff(dst uint16, on bool) Command {
	var state byte
	if on {
		state = 0x01
	}

	return Command{
		Destination:  dst,
		Opcode:       0xed,
		Vendor:       VendorSkytone,
		Data:         []byte{state},
		PlaintextLen: 15,
	}
}

// GroupOnOff builds a group on/off command (opcode 0xe7).
//
// This is an alternative on/off specifically for group control.
// State uses 0x0e = ON, 0x0d = OFF. The group address appears
// twice in the payload.
func GroupOnOff(group uint16, on bool) Command {
	var state byte
	if on {
		state = 0x0e
	} else {
		state = 0x0d
	}

	data := make([]byte, 10)
	data[0] = state
	data[1] = 0x00
	data[2] = 0x10
	data[3] = 0x00
	data[4] = 0x00
	data[5] = 0x00
	byteutil.PutLE16(data[6:8], group)
	data[8] = 0x00

	return Command{
		Destination:  group,
		Opcode:       0xe7,
		Vendor:       VendorSkytone,
		Data:         data,
		PlaintextLen: 15,
	}
}
