package command

import "github.com/tcslater/pigsydust/protocol"

// SetGroupMembership builds a set-group-membership command (opcode 0xEF,
// 15-byte plaintext).
//
//	grp_count(1) || gw_mac5(1) || grp_low[0..N-1] || zero_pad
//
// This command sets the **complete** group membership list for the target
// device — it is NOT add/remove. Passing groups=[2] assigns the device to
// group 2 ONLY, removing it from any other groups. Passing an empty slice
// removes all group memberships.
//
// gwMAC5 is the last byte of the connected node's MAC, required for
// firmware validation.
//
// On success, the device responds with a 0xEE notification mirroring the
// group count.
func SetGroupMembership(dst uint16, groupLowBytes []byte, gwMAC5 byte) Command {
	data := make([]byte, 2+len(groupLowBytes))
	data[0] = byte(len(groupLowBytes))
	data[1] = gwMAC5
	copy(data[2:], groupLowBytes)
	return Command{
		Destination:  dst,
		Opcode:       protocol.OpSetGroup,
		Vendor:       protocol.VendorSkytone,
		Data:         data,
		PlaintextLen: 15,
	}
}

// QueryGroupMembership builds a group-membership query (opcode 0xD7,
// 10-byte plaintext). The target device responds with a 0xD4 notification
// carrying its group list.
//
// Vendor is [protocol.VendorSkytoneAlt] (0x0211).
func QueryGroupMembership(dst uint16) Command {
	return Command{
		Destination:  dst,
		Opcode:       protocol.OpQueryGroup,
		Vendor:       protocol.VendorSkytoneAlt,
		Data:         []byte{0x00, 0x00, 0x00},
		PlaintextLen: 10,
	}
}

// ProbeGroup builds a group-address probe (opcode 0xDD, 10-byte plaintext).
// If any device is a member of the given group, it responds with a 0xD4
// notification. No response means the address is free.
//
// Vendor is [protocol.VendorSkytoneAlt] (0x0211).
func ProbeGroup(groupAddr uint16) Command {
	return Command{
		Destination:  groupAddr,
		Opcode:       protocol.OpProbeGroup,
		Vendor:       protocol.VendorSkytoneAlt,
		Data:         []byte{0x0A, 0x01},
		PlaintextLen: 10,
	}
}
