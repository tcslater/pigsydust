package command

// SetGroupMembership builds a set group membership command (opcode 0xef).
//
// This sets the device's complete group membership list — it is NOT
// an add/remove operation. Sending groups=[2] assigns the device to
// group 2 ONLY, removing it from all other groups.
//
// gwMAC5 is the last byte of the connected gateway's MAC address,
// required for firmware validation.
func SetGroupMembership(dst uint16, groups []uint8, gwMAC5 byte) Command {
	data := make([]byte, 2+len(groups))
	data[0] = byte(len(groups))
	data[1] = gwMAC5
	for i, g := range groups {
		data[2+i] = g
	}

	return Command{
		Destination:  dst,
		Opcode:       0xef,
		Vendor:       VendorSkytone,
		Data:         data,
		PlaintextLen: 15,
	}
}

// QueryGroupMembership builds a group membership query (opcode 0xd7).
//
// The target device responds with a 0xd4 notification containing
// its group list. Uses vendor 0x0211.
func QueryGroupMembership(dst uint16) Command {
	return Command{
		Destination:  dst,
		Opcode:       0xd7,
		Vendor:       VendorSkytoneAlt,
		Data:         []byte{0x00, 0x00, 0x00},
		PlaintextLen: 10,
	}
}

// ProbeGroup builds a group address probe (opcode 0xdd).
//
// Tests whether a group address is in use. If any device is a member,
// it responds with a 0xd4 notification. No response means the address
// is free. Uses vendor 0x0211.
func ProbeGroup(group uint16) Command {
	return Command{
		Destination:  group,
		Opcode:       0xdd,
		Vendor:       VendorSkytoneAlt,
		Data:         []byte{0x0a, 0x01},
		PlaintextLen: 10,
	}
}
