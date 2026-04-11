package command

import "github.com/tcslater/pigsydust/internal/byteutil"

// WriteAlarm builds the two fragment commands (opcode 0xcc) for creating
// or updating an alarm record.
//
// The alarmBytes parameter must be a 16-byte alarm record (see schedule.AlarmRecord).
// Returns exactly two commands that must be sent in order.
func WriteAlarm(alarmBytes [16]byte) [2]Command {
	// Fragment 0: byte 0x00 + alarm[0..8] = 10 data bytes
	frag0Data := make([]byte, 10)
	frag0Data[0] = 0x00 // fragment index
	copy(frag0Data[1:], alarmBytes[0:9])

	// Fragment 1: byte 0x01 + alarm[9..15] + xor checksum = 9 data bytes
	frag1Data := make([]byte, 9)
	frag1Data[0] = 0x01 // fragment index
	copy(frag1Data[1:8], alarmBytes[9:16])
	frag1Data[8] = byteutil.XORFold(alarmBytes[:])

	return [2]Command{
		{
			Destination:  AddrScheduleCoordinator,
			Opcode:       0xcc,
			Vendor:       VendorSkytone,
			Data:         frag0Data,
			PlaintextLen: 15,
		},
		{
			Destination:  AddrScheduleCoordinator,
			Opcode:       0xcc,
			Vendor:       VendorSkytone,
			Data:         frag1Data,
			PlaintextLen: 15,
		},
	}
}

// QueryAlarm builds an alarm slot query (opcode 0xcd).
//
// startSlot is the scan-from cursor (0-indexed). The coordinator returns
// the first occupied slot at or after this position.
// target filters by device/group address; use 0 for all alarms.
// gwMAC5 is the last byte of the connected gateway's MAC.
//
// Response is two 0xc2 notifications per occupied slot. An actual_slot
// of 0xff signals end-of-list.
func QueryAlarm(startSlot uint8, gwMAC5 byte, target uint16) Command {
	data := make([]byte, 5)
	data[0] = startSlot
	data[1] = gwMAC5
	data[2] = 0x00
	byteutil.PutLE16(data[3:5], target)

	return Command{
		Destination:  AddrScheduleCoordinator,
		Opcode:       0xcd,
		Vendor:       VendorSkytone,
		Data:         data,
		PlaintextLen: 15,
	}
}

// DeleteAlarm builds an alarm delete command (opcode 0xce).
//
// slot is the slot index (obtained from SlotQuery after create,
// or from QueryAlarm walk). gwMAC5 is the gateway cookie.
func DeleteAlarm(slot uint8, gwMAC5 byte) Command {
	return Command{
		Destination:  AddrScheduleCoordinator,
		Opcode:       0xce,
		Vendor:       VendorSkytone,
		Data:         []byte{slot, gwMAC5, 0x00},
		PlaintextLen: 15,
	}
}

// SlotQuery builds a slot assignment query (opcode 0xf0).
//
// Sent after writing an alarm via WriteAlarm to learn which slot was assigned.
// gwMAC5 is the gateway cookie.
//
// Response is a 0xd3 notification with the assigned slot index.
func SlotQuery(gwMAC5 byte) Command {
	return Command{
		Destination:  AddrScheduleCoordinator,
		Opcode:       0xf0,
		Vendor:       VendorSkytone,
		Data:         []byte{gwMAC5, 0x00},
		PlaintextLen: 15,
	}
}
