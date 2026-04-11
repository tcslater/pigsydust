package pigsydust

import "fmt"

// Address is a 16-bit mesh address identifying a device, group, or broadcast target.
type Address uint16

const (
	// AddressBroadcast is the full broadcast address used by on/off and time sync.
	AddressBroadcast Address = 0xFFFF

	// AddressBroadcastPoll is the broadcast address used by status polling.
	AddressBroadcastPoll Address = 0x7FFF

	// AddressScheduleCoordinator is the address that receives all alarm operations.
	AddressScheduleCoordinator Address = 0x0030
)

// GroupAddress returns the mesh address for the given group ID.
// Group addresses are encoded as 0x8000 | id.
func GroupAddress(id uint8) Address {
	return Address(0x8000 | uint16(id))
}

// IsGroup reports whether a is a group address (bit 15 set, not broadcast).
func (a Address) IsGroup() bool {
	return a&0x8000 != 0 && a != AddressBroadcast
}

// IsIndividual reports whether a is an individual device address (0x0001-0x7FFF).
func (a Address) IsIndividual() bool {
	return a > 0 && a <= 0x7FFF
}

// GroupID returns the group ID if a is a group address.
// The second return value is false if a is not a group address.
func (a Address) GroupID() (uint8, bool) {
	if !a.IsGroup() {
		return 0, false
	}
	return uint8(a & 0xFF), true
}

func (a Address) String() string {
	if a == AddressBroadcast {
		return "broadcast"
	}
	if a == AddressBroadcastPoll {
		return "broadcast-poll"
	}
	if a == AddressScheduleCoordinator {
		return "schedule-coordinator"
	}
	if a.IsGroup() {
		id, _ := a.GroupID()
		return fmt.Sprintf("group-%d", id)
	}
	return fmt.Sprintf("device-%d", a)
}
