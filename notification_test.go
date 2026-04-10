package piggsydust

import (
	"encoding/hex"
	"testing"
)

func TestParseNotificationWire(t *testing.T) {
	raw := make([]byte, 20)
	raw[0] = 0x01 // sno[0]
	raw[1] = 0x02 // sno[1]
	raw[2] = 0x03 // sno[2]
	raw[3] = 0x05 // src_addr lo
	raw[4] = 0x00 // src_addr hi
	raw[5] = 0xAA // tag[0]
	raw[6] = 0xBB // tag[1]
	// ciphertext: bytes 7-19

	sno, srcAddr, tag, ciphertext, err := ParseNotificationWire(raw)
	if err != nil {
		t.Fatal(err)
	}

	if sno != [3]byte{0x01, 0x02, 0x03} {
		t.Errorf("sno: got %x", sno)
	}
	if srcAddr != 0x0005 {
		t.Errorf("srcAddr: got 0x%04x, want 0x0005", srcAddr)
	}
	if tag != [2]byte{0xAA, 0xBB} {
		t.Errorf("tag: got %x", tag)
	}
	if len(ciphertext) != 13 {
		t.Errorf("ciphertext len: got %d, want 13", len(ciphertext))
	}
}

func TestParseNotificationWire_WrongLength(t *testing.T) {
	_, _, _, _, err := ParseNotificationWire(make([]byte, 19))
	if err == nil {
		t.Error("expected error for 19-byte packet")
	}
}

func TestParseNotification(t *testing.T) {
	plaintext := []byte{0xdb, 0x11, 0x02, 0x00, 0x01, 0x02, 0x45, 0xFF, 0xEE, 0xDD, 0xCC, 0x03, 0x01}

	n, err := ParseNotification(Address(5), plaintext)
	if err != nil {
		t.Fatal(err)
	}

	if n.Source != Address(5) {
		t.Errorf("Source: got %d, want 5", n.Source)
	}
	if n.Opcode != 0xdb {
		t.Errorf("Opcode: got 0x%02x, want 0xdb", n.Opcode)
	}
	if n.Vendor != 0x0211 {
		t.Errorf("Vendor: got 0x%04x, want 0x0211", n.Vendor)
	}
}

func TestParseDeviceStatus(t *testing.T) {
	n := Notification{
		Source:  Address(5),
		Opcode:  0xdb,
		Vendor:  0x0211,
		Payload: []byte{0x00, 0x01, 0x02, 0x45, 0xFF, 0xEE, 0xDD, 0xCC, 0x03, 0x01},
	}

	ds, err := ParseDeviceStatus(n)
	if err != nil {
		t.Fatal(err)
	}

	if ds.Address != Address(5) {
		t.Errorf("Address: got %d", ds.Address)
	}
	if ds.DeviceType != DeviceTypeLeaf {
		t.Errorf("DeviceType: got 0x%02x, want 0x45", ds.DeviceType)
	}
	if ds.MAC[5] != 0xFF || ds.MAC[4] != 0xEE || ds.MAC[3] != 0xDD || ds.MAC[2] != 0xCC {
		t.Errorf("MAC: got %s", ds.MAC)
	}
	if !ds.On {
		t.Error("On: got false, want true")
	}
}

func TestParseDeviceStatus_RejectsDC(t *testing.T) {
	n := Notification{Opcode: 0xdc, Payload: make([]byte, 10)}
	_, err := ParseDeviceStatus(n)
	if err == nil {
		t.Error("expected error for 0xdc (should use ParseBroadcastStatus)")
	}
}

func TestParseBroadcastStatus_TwoDevices(t *testing.T) {
	// Real capture: Laundry(94) OFF, Verandah(251) OFF
	payload, _ := hex.DecodeString("5ec20080fb6b00800000")

	n := Notification{Opcode: 0xdc, Vendor: 0x0211, Payload: payload}
	devices, err := ParseBroadcastStatus(n)
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) != 2 {
		t.Fatalf("devices: got %d, want 2", len(devices))
	}

	// Device A: Laundry, address 94 (0x5e), OFF
	if devices[0].Address != Address(0x5e) {
		t.Errorf("device A address: got %d, want 94", devices[0].Address)
	}
	if devices[0].Brightness != 0x00 {
		t.Errorf("device A brightness: got 0x%02x, want 0x00 (OFF)", devices[0].Brightness)
	}

	// Device B: Verandah, address 251 (0xfb), OFF
	if devices[1].Address != Address(0xfb) {
		t.Errorf("device B address: got %d, want 251", devices[1].Address)
	}
	if devices[1].Brightness != 0x00 {
		t.Errorf("device B brightness: got 0x%02x, want 0x00 (OFF)", devices[1].Brightness)
	}
}

func TestParseBroadcastStatus_TwoDevicesMixed(t *testing.T) {
	// Real capture: Kitchen(47) ON, Flood Light(48) OFF
	payload, _ := hex.DecodeString("2f8a649030ed00b00000")

	n := Notification{Opcode: 0xdc, Vendor: 0x0211, Payload: payload}
	devices, err := ParseBroadcastStatus(n)
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) != 2 {
		t.Fatalf("devices: got %d, want 2", len(devices))
	}

	if devices[0].Address != Address(0x2f) {
		t.Errorf("device A address: got %d, want 47", devices[0].Address)
	}
	if devices[0].Brightness != 0x64 {
		t.Errorf("device A brightness: got 0x%02x, want 0x64 (ON)", devices[0].Brightness)
	}

	if devices[1].Address != Address(0x30) {
		t.Errorf("device B address: got %d, want 48", devices[1].Address)
	}
	if devices[1].Brightness != 0x00 {
		t.Errorf("device B brightness: got 0x%02x, want 0x00 (OFF)", devices[1].Brightness)
	}
}

func TestParseBroadcastStatus_SingleDevice(t *testing.T) {
	// Real capture: Dining(144) ON - unsolicited toggle, second slot zeroed
	payload, _ := hex.DecodeString("90f36480000000000000")

	n := Notification{Opcode: 0xdc, Vendor: 0x0211, Payload: payload}
	devices, err := ParseBroadcastStatus(n)
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) != 1 {
		t.Fatalf("devices: got %d, want 1", len(devices))
	}

	if devices[0].Address != Address(0x90) {
		t.Errorf("address: got %d, want 144", devices[0].Address)
	}
	if devices[0].Brightness != 0x64 {
		t.Errorf("brightness: got 0x%02x, want 0x64 (ON)", devices[0].Brightness)
	}
}

func TestParseBroadcastStatus_SingleDeviceOff(t *testing.T) {
	// Real capture: Dining(144) OFF
	payload, _ := hex.DecodeString("90fe0090000000000000")

	n := Notification{Opcode: 0xdc, Vendor: 0x0211, Payload: payload}
	devices, err := ParseBroadcastStatus(n)
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) != 1 {
		t.Fatalf("devices: got %d, want 1", len(devices))
	}

	if devices[0].Address != Address(0x90) {
		t.Errorf("address: got %d, want 144", devices[0].Address)
	}
	if devices[0].Brightness != 0x00 {
		t.Errorf("brightness: got 0x%02x, want 0x00 (OFF)", devices[0].Brightness)
	}
}

func TestParseGroupMembership(t *testing.T) {
	n := Notification{
		Source:  Address(5),
		Opcode:  0xd4,
		Vendor:  0x0211,
		Payload: []byte{0x01, 0x02, 0x03, 0xff, 0xff},
	}

	gm, err := ParseGroupMembership(n)
	if err != nil {
		t.Fatal(err)
	}

	if len(gm.Groups) != 3 {
		t.Fatalf("Groups: got %d, want 3", len(gm.Groups))
	}
	if gm.Groups[0] != 1 || gm.Groups[1] != 2 || gm.Groups[2] != 3 {
		t.Errorf("Groups: got %v, want [1 2 3]", gm.Groups)
	}
}

func TestParseLEDState(t *testing.T) {
	n := Notification{
		Source:  Address(1),
		Opcode:  0xd3,
		Vendor:  0x6969,
		Payload: []byte{0x94, 0x10, 0xa0, 0x12, 0xb6, 0x0f},
	}

	ls, err := ParseLEDState(n)
	if err != nil {
		t.Fatal(err)
	}

	if !ls.BlueOn {
		t.Error("BlueOn: got false, want true")
	}
	if ls.OrangeLevel != 15 {
		t.Errorf("OrangeLevel: got %d, want 15", ls.OrangeLevel)
	}
}
