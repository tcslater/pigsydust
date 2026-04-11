package pigsydust

import (
	"fmt"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

// Notification represents a decoded notification from a mesh device.
type Notification struct {
	// Source is the mesh address of the device that sent the notification.
	Source Address

	// Opcode is the raw opcode byte from the notification payload.
	Opcode byte

	// Vendor is the 16-bit vendor ID.
	Vendor uint16

	// Payload is the opcode-specific data after the vendor bytes.
	Payload []byte
}

// DeviceStatus holds the decoded state of a mesh device from a
// unicast 0xdb poll response. See [ParseDeviceStatus].
type DeviceStatus struct {
	Address       Address
	MAC           MACAddress
	ProductRev    byte
	ProductClass  byte
	DeviceType    DeviceType
	RoutingMetric byte
	On            bool
}

// BroadcastDeviceStatus holds the compact state of a mesh device from
// a 0xdc broadcast notification. Each 0xdc notification packs up to
// two device statuses. See [ParseBroadcastStatus].
type BroadcastDeviceStatus struct {
	Address       Address
	RoutingMetric byte
	Brightness    byte // 0x00=off, 0x64=on at 100%
	Flags         byte
}

// GroupMembership holds a device's group list, received in response
// to group queries (0xd7) and probes (0xdd).
type GroupMembership struct {
	Address Address
	Groups  []uint8
}

// LEDState holds the LED indicator state of a device, received
// in response to LED queries (0xd9).
type LEDState struct {
	Address     Address
	BlueOn      bool
	OrangeLevel uint8 // 0-15
}

// ParseNotification decodes a decrypted notification plaintext into
// a [Notification]. The plaintext must be at least 3 bytes (opcode + vendor).
func ParseNotification(srcAddr Address, plaintext []byte) (Notification, error) {
	if len(plaintext) < 3 {
		return Notification{}, fmt.Errorf("%w: notification too short (%d bytes)", ErrInvalidPacket, len(plaintext))
	}

	n := Notification{
		Source: srcAddr,
		Opcode: plaintext[0],
		Vendor: byteutil.LE16(plaintext[1:3]),
	}
	if len(plaintext) > 3 {
		n.Payload = plaintext[3:]
	}

	return n, nil
}

// ParseDeviceStatus extracts a [DeviceStatus] from a 0xdb unicast
// poll response. For 0xdc broadcast notifications, use [ParseBroadcastStatus].
func ParseDeviceStatus(n Notification) (DeviceStatus, error) {
	if n.Opcode != 0xdb {
		return DeviceStatus{}, fmt.Errorf("%w: expected opcode 0xdb, got 0x%02x", ErrInvalidPacket, n.Opcode)
	}
	if len(n.Payload) < 10 {
		return DeviceStatus{}, fmt.Errorf("%w: status payload too short (%d bytes)", ErrInvalidPacket, len(n.Payload))
	}

	ds := DeviceStatus{
		Address:      n.Source,
		ProductRev:   n.Payload[1],
		ProductClass: n.Payload[2],
		DeviceType:   DeviceType(n.Payload[3]),
	}

	// MAC bytes at payload[4:8] are [5,4,3,2] in little-endian order.
	ds.MAC[5] = n.Payload[4]
	ds.MAC[4] = n.Payload[5]
	ds.MAC[3] = n.Payload[6]
	ds.MAC[2] = n.Payload[7]

	ds.RoutingMetric = n.Payload[8]
	ds.On = n.Payload[9] != 0

	return ds, nil
}

// ParseBroadcastStatus extracts up to two [BroadcastDeviceStatus] values
// from a 0xdc broadcast notification.
//
// 0xdc notifications are sent in a burst after a 0xc5 status query or
// SetUTC time sync, and also as unsolicited events when a device is
// physically toggled. Each notification packs two 4-byte device slots:
//
//	slot = address(1) | routing_metric(1) | brightness(1) | flags(1)
//
// The second slot is zeroed if only one device is reported.
func ParseBroadcastStatus(n Notification) ([]BroadcastDeviceStatus, error) {
	if n.Opcode != 0xdc {
		return nil, fmt.Errorf("%w: expected opcode 0xdc, got 0x%02x", ErrInvalidPacket, n.Opcode)
	}
	if len(n.Payload) < 8 {
		return nil, fmt.Errorf("%w: broadcast status payload too short (%d bytes)", ErrInvalidPacket, len(n.Payload))
	}

	var out []BroadcastDeviceStatus

	// Slot A: payload[0..3]
	if n.Payload[0] != 0x00 {
		out = append(out, BroadcastDeviceStatus{
			Address:       Address(n.Payload[0]),
			RoutingMetric: n.Payload[1],
			Brightness:    n.Payload[2],
			Flags:         n.Payload[3],
		})
	}

	// Slot B: payload[4..7]
	if n.Payload[4] != 0x00 {
		out = append(out, BroadcastDeviceStatus{
			Address:       Address(n.Payload[4]),
			RoutingMetric: n.Payload[5],
			Brightness:    n.Payload[6],
			Flags:         n.Payload[7],
		})
	}

	return out, nil
}

// ParseGroupMembership extracts a [GroupMembership] from a 0xd4 notification.
func ParseGroupMembership(n Notification) (GroupMembership, error) {
	if n.Opcode != 0xd4 {
		return GroupMembership{}, fmt.Errorf("%w: expected opcode 0xd4, got 0x%02x", ErrInvalidPacket, n.Opcode)
	}

	gm := GroupMembership{Address: n.Source}

	// Group low bytes start at payload[0], terminated by 0x00 or 0xff.
	for _, b := range n.Payload {
		if b == 0x00 || b == 0xff {
			break
		}
		gm.Groups = append(gm.Groups, b)
	}

	return gm, nil
}

// ParseLEDState extracts an [LEDState] from a 0xd3 notification
// received in LED query context.
func ParseLEDState(n Notification) (LEDState, error) {
	if n.Opcode != 0xd3 {
		return LEDState{}, fmt.Errorf("%w: expected opcode 0xd3, got 0x%02x", ErrInvalidPacket, n.Opcode)
	}
	// LED state: payload[2]=b_ch, payload[3]=b_lvl, payload[4]=o_ch, payload[5]=o_lvl
	if len(n.Payload) < 6 {
		return LEDState{}, fmt.Errorf("%w: LED payload too short (%d bytes)", ErrInvalidPacket, len(n.Payload))
	}

	ls := LEDState{
		Address:     n.Source,
		BlueOn:      n.Payload[3] != 0,
		OrangeLevel: n.Payload[5] & 0x0f,
	}

	return ls, nil
}

// ParseAlarmFragment extracts an alarm query fragment from a 0xc2 notification.
// Returns the slot index, fragment index (0 or 1), and fragment data.
// A slot value of 0xff indicates end-of-list.
func ParseAlarmFragment(n Notification) (slot uint8, fragData []byte, err error) {
	if n.Opcode != 0xc2 {
		return 0, nil, fmt.Errorf("%w: expected opcode 0xc2, got 0x%02x", ErrInvalidPacket, n.Opcode)
	}
	if len(n.Payload) < 1 {
		return 0, nil, fmt.Errorf("%w: alarm fragment payload empty", ErrInvalidPacket)
	}

	slot = n.Payload[0]
	if len(n.Payload) > 1 {
		fragData = n.Payload[1:]
	}

	return slot, fragData, nil
}

// ParseSlotAssignment extracts the assigned slot index from a 0xd3 notification
// received in slot query context (after writing an alarm).
func ParseSlotAssignment(n Notification) (slot uint8, err error) {
	if n.Opcode != 0xd3 {
		return 0, fmt.Errorf("%w: expected opcode 0xd3, got 0x%02x", ErrInvalidPacket, n.Opcode)
	}
	// Format: echo(1) || 0x10 0x04 || slot(1)
	if len(n.Payload) < 4 {
		return 0, fmt.Errorf("%w: slot assignment payload too short (%d bytes)", ErrInvalidPacket, len(n.Payload))
	}

	return n.Payload[3], nil
}

// ParseGroupACK extracts the group count from a 0xee notification.
func ParseGroupACK(n Notification) (groupCount uint8, err error) {
	if n.Opcode != 0xee {
		return 0, fmt.Errorf("%w: expected opcode 0xee, got 0x%02x", ErrInvalidPacket, n.Opcode)
	}
	if len(n.Payload) < 1 {
		return 0, fmt.Errorf("%w: group ACK payload empty", ErrInvalidPacket)
	}

	return n.Payload[0], nil
}

// ParseNotificationWire extracts the sequence number, source address,
// tag, and ciphertext from a raw 20-byte notification wire packet.
//
// Wire format: sno(3) || src_addr(2 LE) || tag(2) || ciphertext(13)
func ParseNotificationWire(raw []byte) (sno [3]byte, srcAddr uint16, tag [2]byte, ciphertext []byte, err error) {
	if len(raw) != 20 {
		return sno, 0, tag, nil, fmt.Errorf("%w: notification must be 20 bytes, got %d", ErrInvalidPacket, len(raw))
	}

	copy(sno[:], raw[0:3])
	srcAddr = byteutil.LE16(raw[3:5])
	copy(tag[:], raw[5:7])
	ciphertext = make([]byte, 13)
	copy(ciphertext, raw[7:20])

	return sno, srcAddr, tag, ciphertext, nil
}
