package command

// LEDSetBlue builds an LED indicator command (opcode 0xff) to control
// the blue channel.
//
// The blue channel is binary on/off (no PWM dimming).
// Each packet must update exactly one channel — the orange channel
// bytes are zeroed.
func LEDSetBlue(dst uint16, on bool) Command {
	var level byte
	if on {
		level = 0x12
	}
	return Command{
		Destination:  dst,
		Opcode:       0xff,
		Vendor:       VendorSkytone,
		Data:         []byte{0xa0, level, 0x00, 0x00},
		PlaintextLen: 15,
	}
}

// LEDSetOrange builds an LED indicator command (opcode 0xff) to control
// the orange channel.
//
// The orange channel is PWM-dimmable; level is the brightness (0-15).
// A level of 0 turns the orange LED off.
// Each packet must update exactly one channel — the blue channel
// bytes are zeroed.
func LEDSetOrange(dst uint16, level uint8) Command {
	return Command{
		Destination:  dst,
		Opcode:       0xff,
		Vendor:       VendorSkytone,
		Data:         []byte{0x00, 0x00, 0xff, level & 0x0f},
		PlaintextLen: 15,
	}
}

// LEDQuery builds an LED indicator query (opcode 0xd9).
//
// This uses vendor 0x696b (unique to this opcode). The gwMAC5 parameter
// is the last byte of the connected gateway's MAC address — it acts as a
// relay routing tag. Sending the wrong value causes a silent timeout.
//
// Required query sequence:
//  1. Send StatusPoll to the target device
//  2. Wait for the 0xdb response
//  3. Wait ~210ms
//  4. Send this LEDQuery
//  5. 0xd3 response arrives within ~60ms
func LEDQuery(dst uint16, gwMAC5 byte) Command {
	return Command{
		Destination:  dst,
		Opcode:       0xd9,
		Vendor:       VendorLEDQuery,
		Data:         []byte{gwMAC5, 0x00},
		PlaintextLen: 10,
	}
}

// FindMe builds a find-me LED flash command (opcode 0xf5).
//
// When start is true, the device blinks for 15 seconds using the
// currently configured LED colour. When false, blinking stops.
func FindMe(dst uint16, start bool) Command {
	var mode, duration byte
	if start {
		mode = 0x03
		duration = 0x0f
	}
	return Command{
		Destination:  dst,
		Opcode:       0xf5,
		Vendor:       VendorSkytone,
		Data:         []byte{mode, duration},
		PlaintextLen: 15,
	}
}
