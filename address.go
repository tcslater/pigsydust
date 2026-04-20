package pigsydust

// Address is a 16-bit mesh address.
//
// The address space is partitioned:
//
//   - 0x0001 – 0x7FFE: individual device addresses
//   - 0x7FFF:          broadcast-poll (used by status polling)
//   - 0x8000 | id:     group address (e.g. 0x8001 = group 1)
//   - 0xFFFF:          full broadcast (on/off, time sync)
//   - 0x0030:          schedule coordinator (receives alarm ops)
type Address uint16

// Well-known mesh addresses.
const (
	AddrBroadcast       Address = 0xFFFF
	AddrBroadcastPoll   Address = 0x7FFF
	AddrScheduleCoord   Address = 0x0030
	groupAddrBit        uint16  = 0x8000
)

// IsGroup reports whether a is a group address (high bit set).
func (a Address) IsGroup() bool {
	return uint16(a)&groupAddrBit != 0 && a != AddrBroadcast
}

// IsIndividual reports whether a is a non-broadcast individual address.
func (a Address) IsIndividual() bool {
	return a != 0 && a < 0x7FFF
}

// GroupID returns the group ID (low byte) if a is a group address. Returns 0
// for non-group addresses.
func (a Address) GroupID() uint8 {
	if !a.IsGroup() {
		return 0
	}
	return uint8(a & 0x00FF)
}

// GroupAddress returns the 16-bit address for group id.
func GroupAddress(id uint8) Address {
	return Address(groupAddrBit | uint16(id))
}
