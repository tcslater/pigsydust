package pigsydust

import (
	"encoding/binary"
	"fmt"

	"github.com/tcslater/pigsydust/crypto"
	"github.com/tcslater/pigsydust/protocol"
)

// Notification is a decrypted notification frame from the mesh.
type Notification struct {
	// Source is the device mesh address the notification came from.
	// For 0xDC broadcast status frames, wire src_addr is 0x0000 — the
	// real device addresses are embedded in the payload.
	Source uint16
	// Opcode is the plaintext opcode byte.
	Opcode byte
	// Vendor is the plaintext vendor ID (little-endian on the wire).
	Vendor uint16
	// Payload is the remaining plaintext after opcode+vendor (opcode-specific).
	Payload []byte
}

// ParseNotificationWire extracts the four wire fields from a raw 20-byte
// notification:
//
//	sno(3) || src_addr(2 LE) || tag(2) || ciphertext(13)
func ParseNotificationWire(raw []byte) (sno [3]byte, srcAddr uint16, tag [2]byte, ciphertext []byte, err error) {
	if len(raw) != 20 {
		err = fmt.Errorf("pigsydust: notification must be 20 bytes, got %d", len(raw))
		return
	}
	copy(sno[:], raw[0:3])
	srcAddr = binary.LittleEndian.Uint16(raw[3:5])
	copy(tag[:], raw[5:7])
	ciphertext = raw[7:20]
	return
}

// DecryptNotification decrypts a raw 20-byte notification packet using the
// session key and the connected gateway's MAC. Returns [ErrShortPacket] if
// the plaintext is truncated, or a crypto error on tag mismatch.
func DecryptNotification(sk [16]byte, gwMAC MACAddress, raw []byte) (Notification, error) {
	var n Notification
	sno, srcAddr, tag, ct, err := ParseNotificationWire(raw)
	if err != nil {
		return n, err
	}
	nonce := crypto.NotificationNonce([6]byte(gwMAC), sno, srcAddr)
	pt, err := crypto.Decrypt(sk, nonce, tag, ct)
	if err != nil {
		return n, err
	}
	if len(pt) < 3 {
		return n, fmt.Errorf("%w: notification plaintext %d bytes", ErrShortPacket, len(pt))
	}
	n = Notification{
		Source:  srcAddr,
		Opcode:  pt[0],
		Vendor:  binary.LittleEndian.Uint16(pt[1:3]),
		Payload: append([]byte(nil), pt[3:]...),
	}
	return n, nil
}

// StatusFlags is the decoded bit layout of the packed status byte found in
// both advertisement byte 8 and decrypted 0xDB status payloads.
type StatusFlags struct {
	Online   bool  // bit 0
	AlarmDev bool  // bit 1
	Version  uint8 // bits 2-7 (6-bit firmware version)
}

// ParseStatusFlags decomposes a packed status byte into its fields.
func ParseStatusFlags(b byte) StatusFlags {
	return StatusFlags{
		Online:   b&0x01 != 0,
		AlarmDev: b&0x02 != 0,
		Version:  b >> 2,
	}
}

// DeviceStatus is the decoded status of a single mesh device.
type DeviceStatus struct {
	Address        uint16
	On             bool
	MAC            MACAddress
	RoutingMetric  uint8
	DeviceType     byte
	DeviceSubtype  byte
	StatusByte     byte
	StatusFlags    StatusFlags
}

// DeviceClass resolves the wire type/subtype bytes to a canonical device
// class, or [protocol.DeviceClassUnknown] if the pair isn't in the table.
func (s DeviceStatus) DeviceClass() protocol.DeviceClass {
	return protocol.DeviceClassLookup(s.DeviceType, s.DeviceSubtype)
}

// ParseDeviceStatus parses a 0xDB notification (unicast status response).
//
// 0xDB payload (10 bytes after opcode + vendor):
//
//	padding(1) || type(1) || stype(1) || status_byte(1) ||
//	mac[5:4:3:2](4) || routing_metric(1) || on_off(1)
//
// type and stype are wire-halved; status_byte uses the same layout as the
// advertisement packed byte.
func ParseDeviceStatus(n Notification) (DeviceStatus, error) {
	var ds DeviceStatus
	if n.Opcode != protocol.OpNotifyStatusPoll {
		return ds, fmt.Errorf("%w: expected 0xDB, got 0x%02X", ErrUnexpectedOpcode, n.Opcode)
	}
	if len(n.Payload) < 10 {
		return ds, fmt.Errorf("%w: 0xDB payload %d bytes", ErrShortPacket, len(n.Payload))
	}
	ds = DeviceStatus{
		Address:       n.Source,
		DeviceType:    n.Payload[1],
		DeviceSubtype: n.Payload[2],
		StatusByte:    n.Payload[3],
		StatusFlags:   ParseStatusFlags(n.Payload[3]),
		RoutingMetric: n.Payload[8],
		On:            n.Payload[9] != 0,
	}
	// MAC bytes 5..2 in payload[4..7] (bytes 0 and 1 are not reported).
	ds.MAC[5] = n.Payload[4]
	ds.MAC[4] = n.Payload[5]
	ds.MAC[3] = n.Payload[6]
	ds.MAC[2] = n.Payload[7]
	return ds, nil
}

// ParseDeviceStatusBroadcast parses a 0xDC notification (broadcast status
// burst or unsolicited state change). The wire src_addr is always 0; real
// device addresses are embedded in the payload. Up to two device statuses
// are packed per notification.
//
// 0xDC payload (10 bytes after opcode + vendor):
//
//	dev_a: addr(1) metric(1) brightness(1) flags(1)
//	dev_b: addr(1) metric(1) brightness(1) flags(1)
//	padding(2)
//
// A zero address in either slot means the slot is empty (unsolicited
// change events populate only slot A).
func ParseDeviceStatusBroadcast(n Notification) ([]DeviceStatus, error) {
	if n.Opcode != protocol.OpNotifyStatusBroadcast {
		return nil, fmt.Errorf("%w: expected 0xDC, got 0x%02X", ErrUnexpectedOpcode, n.Opcode)
	}
	if len(n.Payload) < 8 {
		return nil, fmt.Errorf("%w: 0xDC payload %d bytes", ErrShortPacket, len(n.Payload))
	}

	var out []DeviceStatus
	for _, offset := range []int{0, 4} {
		addr := n.Payload[offset]
		if addr == 0 {
			continue
		}
		flags := n.Payload[offset+3]
		out = append(out, DeviceStatus{
			Address:       uint16(addr),
			RoutingMetric: n.Payload[offset+1],
			On:            n.Payload[offset+2] != 0,
			StatusByte:    flags,
			StatusFlags:   ParseStatusFlags(flags),
		})
	}
	return out, nil
}

// GroupMembership is the set of group low-bytes a device belongs to, as
// reported by a 0xD4 notification (response to 0xD7 query or 0xDD probe).
type GroupMembership struct {
	Groups []byte
}

// ParseGroupMembership parses a 0xD4 notification.
//
//	opcode(0xD4) || vendor(0x0211) || grp_low[0..] || terminator(0x00 or 0xFF)
//
// After the vendor bytes, group low-bytes run until a 0x00 or 0xFF
// terminator is hit or the payload ends.
func ParseGroupMembership(n Notification) (GroupMembership, error) {
	var gm GroupMembership
	if n.Opcode != protocol.OpNotifyGroupResponse {
		return gm, fmt.Errorf("%w: expected 0xD4, got 0x%02X", ErrUnexpectedOpcode, n.Opcode)
	}
	for _, b := range n.Payload {
		if b == 0x00 || b == 0xFF {
			break
		}
		gm.Groups = append(gm.Groups, b)
	}
	return gm, nil
}

// LEDState is the decoded response to an 0xD9 LED indicator query
// (notification opcode 0xD3).
type LEDState struct {
	BlueChannel   byte
	BlueLevel     byte
	OrangeChannel byte
	OrangeLevel   byte
}

// BlueOn reports whether the blue channel is lit.
func (s LEDState) BlueOn() bool { return s.BlueLevel != 0 }

// OrangeOn reports whether the orange channel is lit.
func (s LEDState) OrangeOn() bool { return s.OrangeLevel != 0 }

// ParseLEDState parses the LED-state flavour of a 0xD3 notification.
//
// 0xD3 LED-state payload (after opcode + vendor):
//
//	0x94 0x10 || b_ch(1) b_lvl(1) o_ch(1) o_lvl(1) || tail
//
// The orange channel byte in the response is [protocol.LEDChOrangeInternal]
// (0xB6), not the 0xFF setter value.
func ParseLEDState(n Notification) (LEDState, error) {
	var s LEDState
	if n.Opcode != protocol.OpNotifyLEDOrSlot {
		return s, fmt.Errorf("%w: expected 0xD3, got 0x%02X", ErrUnexpectedOpcode, n.Opcode)
	}
	if len(n.Payload) < 6 {
		return s, fmt.Errorf("%w: 0xD3 LED payload %d bytes", ErrShortPacket, len(n.Payload))
	}
	// payload[0..1] = 0x94 0x10 header (LED-state flavour). Slot-assignment
	// flavour uses a different header layout (see ParseSlotAssignment).
	s.BlueChannel = n.Payload[2]
	s.BlueLevel = n.Payload[3]
	s.OrangeChannel = n.Payload[4]
	s.OrangeLevel = n.Payload[5]
	return s, nil
}

// ParseSlotAssignment parses the slot-assignment flavour of a 0xD3
// notification (response to an [command.SlotQuery]).
//
// 0xD3 slot-assignment payload:
//
//	echo(1) || 0x10 0x04 || slot(1) || zero_pad
//
// Returns the assigned slot index.
func ParseSlotAssignment(n Notification) (byte, error) {
	if n.Opcode != protocol.OpNotifyLEDOrSlot {
		return 0, fmt.Errorf("%w: expected 0xD3, got 0x%02X", ErrUnexpectedOpcode, n.Opcode)
	}
	if len(n.Payload) < 4 {
		return 0, fmt.Errorf("%w: 0xD3 slot payload %d bytes", ErrShortPacket, len(n.Payload))
	}
	return n.Payload[3], nil
}

// AlarmFragment is one fragment of a 0xC2 alarm-query response. Two
// fragments (index 0 and 1) carry a complete 16-byte alarm record together.
type AlarmFragment struct {
	Slot  byte
	Index byte // 0 or 1
	Data  []byte
}

// ParseAlarmFragment parses a 0xC2 notification.
//
//	opcode(0xC2) || vendor(0x6969) || actual_slot(1) || frag_data[9]
//
// An end-of-list sentinel has actual_slot = 0xFF; callers should check
// that before interpreting Data.
func ParseAlarmFragment(n Notification) (AlarmFragment, error) {
	var f AlarmFragment
	if n.Opcode != protocol.OpNotifyAlarmFragment {
		return f, fmt.Errorf("%w: expected 0xC2, got 0x%02X", ErrUnexpectedOpcode, n.Opcode)
	}
	if len(n.Payload) < 1 {
		return f, fmt.Errorf("%w: 0xC2 payload empty", ErrShortPacket)
	}
	f.Slot = n.Payload[0]
	f.Data = append([]byte(nil), n.Payload[1:]...)
	return f, nil
}
