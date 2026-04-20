package pigsydust

import "testing"

// Per spec: MAC bytes 5..2 at data[2..5], type at data[6], stype at data[7],
// status at data[8], mesh_address at data[9], network_id at data[11..15].
var sampleAdv = []byte{
	0x11, 0x02, // company ID prefix (advert re-emits it at 0-1)
	0xEE, 0xDD, 0xCC, 0xBB, // MAC[5..2]
	0x22, 0x18, // type, stype
	0x05,       // status byte (online=1, alarmDev=0, version=1)
	0xEE,       // mesh_address
	0x00,       // padding
	0x78, 0x56, 0x34, 0x12, // network_id LE
}

func TestParseAdvertisementFields(t *testing.T) {
	adv := ParseAdvertisement(sampleAdv)
	if adv == nil {
		t.Fatal("ParseAdvertisement returned nil")
	}
	if adv.MAC[5] != 0xEE || adv.MAC[4] != 0xDD || adv.MAC[3] != 0xCC || adv.MAC[2] != 0xBB {
		t.Errorf("MAC[5..2] = %02x %02x %02x %02x, want EE DD CC BB",
			adv.MAC[5], adv.MAC[4], adv.MAC[3], adv.MAC[2])
	}
	if adv.DeviceType != 0x22 || adv.DeviceSubtype != 0x18 {
		t.Errorf("type/stype = 0x%02x/0x%02x", adv.DeviceType, adv.DeviceSubtype)
	}
	if !adv.StatusFlags.Online {
		t.Error("online flag should be set")
	}
	if adv.StatusFlags.Version != 1 {
		t.Errorf("version = %d, want 1", adv.StatusFlags.Version)
	}
	if adv.MeshAddress != 0xEE {
		t.Errorf("mesh_address = 0x%02x", adv.MeshAddress)
	}
	if adv.NetworkID != 0x12345678 {
		t.Errorf("network_id = 0x%08x, want 0x12345678", adv.NetworkID)
	}
}

func TestParseAdvertisementShortData(t *testing.T) {
	if ParseAdvertisement(sampleAdv[:14]) != nil {
		t.Error("expected nil for 14-byte buffer")
	}
}

func TestParseManufacturerDataRejectsWrongID(t *testing.T) {
	if _, err := ParseManufacturerData(0x004C, sampleAdv); err == nil {
		t.Error("expected error for non-Skytone company ID")
	}
}

func TestParseManufacturerDataOK(t *testing.T) {
	adv, err := ParseManufacturerData(0x0211, sampleAdv)
	if err != nil {
		t.Fatal(err)
	}
	if adv.NetworkID != 0x12345678 {
		t.Errorf("network_id = 0x%08x", adv.NetworkID)
	}
}
