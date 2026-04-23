package command

import "github.com/tcslater/pigsydust/protocol"

// LEDSetBlue builds an LED command for the blue channel (opcode 0xFF,
// 15-byte plaintext). The blue channel is binary — any non-zero level
// lights it.
//
// Each LED packet must touch exactly one channel; the orange bytes are
// zeroed so firmware leaves orange untouched.
func LEDSetBlue(dst uint16, on bool) Command {
	level := byte(0)
	if on {
		level = protocol.LEDBlueOnLevel
	}
	return Command{
		Destination: dst,
		Opcode:      protocol.OpLEDSet,
		Vendor:      protocol.VendorSkytone,
		Data: []byte{
			protocol.LEDChBlueSelect, level,
			0x00, 0x00, // orange untouched
		},
		PlaintextLen: 15,
	}
}

// LEDSetOrange builds an LED command for the orange channel (opcode 0xFF,
// 15-byte plaintext). Orange is PWM-dimmable; level is 0-15 (0 = off).
// Higher bits of level are masked off by the firmware.
//
// Each LED packet must touch exactly one channel; the blue bytes are zeroed
// so firmware leaves blue untouched.
func LEDSetOrange(dst uint16, level byte) Command {
	return Command{
		Destination: dst,
		Opcode:      protocol.OpLEDSet,
		Vendor:      protocol.VendorSkytone,
		Data: []byte{
			0x00, 0x00, // blue untouched
			protocol.LEDChOrangeSelect, level & 0x0F,
		},
		PlaintextLen: 15,
	}
}

// LEDSetPurple builds a combined LED command lighting both channels
// simultaneously (opcode 0xFF, 15-byte plaintext), producing purple.
//
// Warning: sending this latches the firmware into an undefined state that
// survives subsequent single-channel updates. Clear the state with the
// reset sequence (blue-off → orange-off → single-channel commands).
func LEDSetPurple(dst uint16, orangeLevel byte) Command {
	return Command{
		Destination: dst,
		Opcode:      protocol.OpLEDSet,
		Vendor:      protocol.VendorSkytone,
		Data: []byte{
			protocol.LEDChBlueSelect, protocol.LEDBlueOnLevel,
			protocol.LEDChOrangeSelect, orangeLevel & 0x0F,
		},
		PlaintextLen: 15,
	}
}

// LEDQuery builds an LED indicator query (opcode 0xD9, 15-byte plaintext).
// Vendor is [protocol.VendorLEDQuery] (0x696B) — unique to this opcode.
//
// gwMAC5 is the last byte of the connected node's MAC, used as a firmware
// routing tag — a wrong value causes the response to be silently dropped
// with no error.
//
// The query requires a wake-up sequence on dormant devices: send a unicast
// [StatusPoll] first, await the 0xDB wake-up notification, wait ~210 ms,
// then send this query. The 0xD3 response arrives within ~60 ms.
func LEDQuery(dst uint16, gwMAC5 byte) Command {
	return Command{
		Destination: dst,
		Opcode:      protocol.OpLEDQuery,
		Vendor:      protocol.VendorLEDQuery,
		Data: []byte{
			gwMAC5, 0x00,
		},
		PlaintextLen: 15,
	}
}

// FindMe builds a find-me LED flash command (opcode 0xF5, 15-byte
// plaintext). When start is true the device blinks for ~15 seconds using
// the currently configured LED colour; false stops blinking.
func FindMe(dst uint16, start bool) Command {
	var data []byte
	if start {
		data = []byte{protocol.FindMeModeBlink, protocol.FindMeDuration}
	} else {
		data = []byte{0x00, 0x00}
	}
	return Command{
		Destination:  dst,
		Opcode:       protocol.OpFindMe,
		Vendor:       protocol.VendorSkytone,
		Data:         data,
		PlaintextLen: 15,
	}
}
