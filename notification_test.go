package pigsydust

import (
	"testing"

	"github.com/tcslater/pigsydust/protocol"
)

func TestParseStatusFlagsBits(t *testing.T) {
	f := ParseStatusFlags(0x05) // 0b00000101 → online=1, alarmDev=0, version=1
	if !f.Online {
		t.Error("online bit")
	}
	if f.AlarmDev {
		t.Error("alarmDev should be clear")
	}
	if f.Version != 1 {
		t.Errorf("version = %d, want 1", f.Version)
	}

	f = ParseStatusFlags(0xFE) // 0b11111110 → online=0, alarmDev=1, version=63
	if f.Online {
		t.Error("online should be clear")
	}
	if !f.AlarmDev {
		t.Error("alarmDev should be set")
	}
	if f.Version != 0x3F {
		t.Errorf("version = %d, want 63", f.Version)
	}
}

func TestParseNotificationWireLen(t *testing.T) {
	if _, _, _, _, err := ParseNotificationWire(make([]byte, 19)); err == nil {
		t.Error("expected error for 19-byte notification")
	}
	if _, _, _, _, err := ParseNotificationWire(make([]byte, 20)); err != nil {
		t.Errorf("20 bytes should parse: %v", err)
	}
}

func TestParseDeviceStatus(t *testing.T) {
	// payload after opcode+vendor:
	// padding(1) || type(1) || stype(1) || status(1) ||
	// mac[5:4:3:2](4) || routing_metric(1) || on_off(1)
	n := Notification{
		Source:  0x007D,
		Opcode:  protocol.OpNotifyStatusPoll,
		Vendor:  protocol.VendorSkytoneAlt,
		Payload: []byte{0x00, 0x22, 0x18, 0x05, 0x11, 0x22, 0x33, 0x44, 0x07, 0x01},
	}
	ds, err := ParseDeviceStatus(n)
	if err != nil {
		t.Fatal(err)
	}
	if ds.Address != 0x007D {
		t.Errorf("addr = 0x%04x", ds.Address)
	}
	if ds.DeviceType != 0x22 || ds.DeviceSubtype != 0x18 {
		t.Errorf("type/stype mismatch")
	}
	if !ds.On {
		t.Error("on should be true")
	}
	if ds.RoutingMetric != 0x07 {
		t.Errorf("routing metric = %d", ds.RoutingMetric)
	}
	if ds.MAC[5] != 0x11 || ds.MAC[4] != 0x22 || ds.MAC[3] != 0x33 || ds.MAC[2] != 0x44 {
		t.Errorf("MAC bytes misplaced: %v", ds.MAC)
	}
}

func TestParseDeviceStatusWrongOpcode(t *testing.T) {
	n := Notification{Opcode: 0xCC, Payload: make([]byte, 10)}
	if _, err := ParseDeviceStatus(n); err == nil {
		t.Error("expected unexpected-opcode error")
	}
}

func TestParseDeviceStatusBroadcastTwoSlots(t *testing.T) {
	n := Notification{
		Opcode: protocol.OpNotifyStatusBroadcast,
		Payload: []byte{
			0x7D, 0x05, 0x01, 0x05, // slot A
			0x7E, 0x03, 0x00, 0x01, // slot B
			0x00, 0x00,
		},
	}
	out, err := ParseDeviceStatusBroadcast(n)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d devices, want 2", len(out))
	}
	if out[0].Address != 0x007D || !out[0].On {
		t.Errorf("slot A wrong: %+v", out[0])
	}
	if out[1].Address != 0x007E || out[1].On {
		t.Errorf("slot B wrong: %+v", out[1])
	}
}

func TestParseDeviceStatusBroadcastEmptySlotA(t *testing.T) {
	n := Notification{
		Opcode: protocol.OpNotifyStatusBroadcast,
		Payload: []byte{
			0x00, 0x00, 0x00, 0x00, // slot A empty
			0x7E, 0x03, 0x00, 0x01, // slot B
			0x00, 0x00,
		},
	}
	out, _ := ParseDeviceStatusBroadcast(n)
	if len(out) != 1 || out[0].Address != 0x007E {
		t.Errorf("expected only slot B: %+v", out)
	}
}

func TestParseGroupMembershipTerminator(t *testing.T) {
	n := Notification{
		Opcode:  protocol.OpNotifyGroupResponse,
		Payload: []byte{0x02, 0x05, 0x0A, 0xFF, 0x99}, // 0xFF terminates
	}
	gm, err := ParseGroupMembership(n)
	if err != nil {
		t.Fatal(err)
	}
	if len(gm.Groups) != 3 {
		t.Errorf("groups = %v, want 3 entries", gm.Groups)
	}
}

func TestParseLEDState(t *testing.T) {
	n := Notification{
		Opcode: protocol.OpNotifyLEDOrSlot,
		// 0x94 0x10 header, then b_ch, b_lvl, o_ch, o_lvl
		Payload: []byte{0x94, 0x10, 0xA0, 0x12, 0xB6, 0x0F},
	}
	s, err := ParseLEDState(n)
	if err != nil {
		t.Fatal(err)
	}
	if !s.BlueOn() || !s.OrangeOn() {
		t.Errorf("both channels should be on")
	}
	if s.OrangeChannel != protocol.LEDChOrangeInternal {
		t.Errorf("orange channel = 0x%02x, want 0xB6", s.OrangeChannel)
	}
}

func TestParseSlotAssignment(t *testing.T) {
	n := Notification{
		Opcode:  protocol.OpNotifyLEDOrSlot,
		Payload: []byte{0xAA, 0x10, 0x04, 0x07, 0x00},
	}
	slot, err := ParseSlotAssignment(n)
	if err != nil {
		t.Fatal(err)
	}
	if slot != 0x07 {
		t.Errorf("slot = 0x%02x, want 0x07", slot)
	}
}

func TestParseAlarmFragment(t *testing.T) {
	n := Notification{
		Opcode:  protocol.OpNotifyAlarmFragment,
		Payload: []byte{0x05, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
	}
	f, err := ParseAlarmFragment(n)
	if err != nil {
		t.Fatal(err)
	}
	if f.Slot != 0x05 {
		t.Errorf("slot = 0x%02x", f.Slot)
	}
	if len(f.Data) != 9 {
		t.Errorf("data len = %d, want 9", len(f.Data))
	}
}
