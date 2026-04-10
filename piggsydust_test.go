package piggsydust

import "testing"

func TestMACAddress_String(t *testing.T) {
	mac := MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if got := mac.String(); got != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("String(): got %q, want %q", got, "AA:BB:CC:DD:EE:FF")
	}
}

func TestMACAddress_GatewayMAC5(t *testing.T) {
	mac := MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if got := mac.GatewayMAC5(); got != 0xFF {
		t.Errorf("GatewayMAC5(): got 0x%02x, want 0xFF", got)
	}
}

func TestParseMAC(t *testing.T) {
	mac, err := ParseMAC("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatal(err)
	}
	expected := MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if mac != expected {
		t.Errorf("ParseMAC: got %s, want %s", mac, expected)
	}
}

func TestParseMAC_Invalid(t *testing.T) {
	_, err := ParseMAC("not-a-mac")
	if err == nil {
		t.Error("expected error for invalid MAC")
	}
}

func TestParseManufacturerData(t *testing.T) {
	data := make([]byte, 21)
	// MAC bytes at offset 2-5 (LE: [5,4,3,2])
	data[2] = 0xFF // MAC[5]
	data[3] = 0xEE // MAC[4]
	data[4] = 0xDD // MAC[3]
	data[5] = 0xCC // MAC[2]
	// Device type at offset 14
	data[14] = 0x47 // gateway
	// Network ID at offset 17-20 (LE)
	data[17] = 0x01
	data[18] = 0x02
	data[19] = 0x03
	data[20] = 0x04

	ad, err := ParseManufacturerData(0x0211, data)
	if err != nil {
		t.Fatal(err)
	}

	if ad.MAC[5] != 0xFF || ad.MAC[4] != 0xEE || ad.MAC[3] != 0xDD || ad.MAC[2] != 0xCC {
		t.Errorf("MAC: got %s", ad.MAC)
	}
	if ad.DeviceType != DeviceTypeGateway {
		t.Errorf("DeviceType: got 0x%02x, want 0x47", ad.DeviceType)
	}
	if ad.NetworkID != 0x04030201 {
		t.Errorf("NetworkID: got 0x%08x, want 0x04030201", ad.NetworkID)
	}
}

func TestParseManufacturerData_WrongCompanyID(t *testing.T) {
	_, err := ParseManufacturerData(0x9999, make([]byte, 21))
	if err == nil {
		t.Error("expected error for wrong company ID")
	}
}

func TestDeviceType_String(t *testing.T) {
	if DeviceTypeGateway.String() != "gateway" {
		t.Errorf("got %q", DeviceTypeGateway.String())
	}
	if DeviceTypeLeaf.String() != "leaf" {
		t.Errorf("got %q", DeviceTypeLeaf.String())
	}
}
