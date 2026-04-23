package command

import "github.com/tcslater/pigsydust/protocol"

// ScheduleCoordinator is the well-known destination address for all
// schedule operations (0x0030).
const ScheduleCoordinator uint16 = 0x0030

// WriteAlarm builds the two-fragment alarm-write command pair (opcode
// 0xCC, 15-byte plaintext each).
//
//	Fragment 0: 0x00 || alarm[0..8]         (9 alarm bytes)
//	Fragment 1: 0x01 || alarm[9..15] || xor (7 alarm bytes + checksum)
//
// The xor byte is the XOR-fold of all 16 alarm record bytes. See
// [github.com/tcslater/pigsydust/schedule] for record layout.
//
// Callers must send the two returned commands in order and wait for
// acknowledgement before issuing a slot query.
func WriteAlarm(record [16]byte) [2]Command {
	xor := byte(0)
	for _, b := range record {
		xor ^= b
	}

	frag0 := Command{
		Destination: ScheduleCoordinator,
		Opcode:      protocol.OpWriteAlarm,
		Vendor:      protocol.VendorSkytone,
		Data: append(
			[]byte{0x00},
			record[0:9]...,
		),
		PlaintextLen: 15,
	}

	frag1 := Command{
		Destination: ScheduleCoordinator,
		Opcode:      protocol.OpWriteAlarm,
		Vendor:      protocol.VendorSkytone,
		Data: append(
			[]byte{0x01},
			append(append([]byte{}, record[9:16]...), xor)...,
		),
		PlaintextLen: 15,
	}

	return [2]Command{frag0, frag1}
}

// QueryAlarm builds a query command for alarm slots (opcode 0xCD, 15-byte
// plaintext).
//
//	start_slot(1) || gw_mac5(1) || 0x00 || target_lo(1) || target_hi(1)
//
// start is a scan-from cursor (0-indexed). The coordinator returns the
// first occupied slot at or after this position. target filters by device
// or group address (use 0x0000 to list all alarms).
//
// The response is two 0xC2 notifications per occupied slot, carrying the
// 16-byte alarm record in the same fragmentation as [WriteAlarm]. An
// end-of-list response has actual_slot = 0xFF and the remaining data
// zeroed.
func QueryAlarm(startSlot, gwMAC5 byte, target uint16) Command {
	return Command{
		Destination: ScheduleCoordinator,
		Opcode:      protocol.OpQueryAlarm,
		Vendor:      protocol.VendorSkytone,
		Data: []byte{
			startSlot,
			gwMAC5,
			0x00,
			byte(target), byte(target >> 8),
		},
		PlaintextLen: 15,
	}
}

// DeleteAlarm builds a delete command for a single alarm slot (opcode 0xCE,
// 15-byte plaintext).
//
//	slot(1) || gw_mac5(1) || 0x00
//
// slot is the real slot index (from [SlotQuery] after write, or from a
// [QueryAlarm] walk).
func DeleteAlarm(slot, gwMAC5 byte) Command {
	return Command{
		Destination:  ScheduleCoordinator,
		Opcode:       protocol.OpDeleteAlarm,
		Vendor:       protocol.VendorSkytone,
		Data:         []byte{slot, gwMAC5, 0x00},
		PlaintextLen: 15,
	}
}

// SlotQuery builds a slot-assignment query (opcode 0xF0, 15-byte
// plaintext), sent after [WriteAlarm] to learn which slot the coordinator
// assigned.
//
// The response is a 0xD3 notification with the assigned slot in its
// payload.
func SlotQuery(gwMAC5 byte) Command {
	return Command{
		Destination:  ScheduleCoordinator,
		Opcode:       protocol.OpSlotQuery,
		Vendor:       protocol.VendorSkytone,
		Data:         []byte{gwMAC5, 0x00},
		PlaintextLen: 15,
	}
}
