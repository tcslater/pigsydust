package pigsydust

import "testing"

func TestAddressClassification(t *testing.T) {
	cases := []struct {
		addr       Address
		individual bool
		group      bool
	}{
		{0x0001, true, false},
		{0x7FFE, true, false},
		{0x7FFF, false, false}, // broadcast-poll
		{0x8001, false, true},
		{0xFFFE, false, true},
		{AddrBroadcast, false, false},
	}
	for _, c := range cases {
		if got := c.addr.IsIndividual(); got != c.individual {
			t.Errorf("0x%04x IsIndividual = %v, want %v", uint16(c.addr), got, c.individual)
		}
		if got := c.addr.IsGroup(); got != c.group {
			t.Errorf("0x%04x IsGroup = %v, want %v", uint16(c.addr), got, c.group)
		}
	}
}

func TestGroupRoundTrip(t *testing.T) {
	for id := 0; id <= 0xFF; id++ {
		a := GroupAddress(byte(id))
		if !a.IsGroup() {
			t.Errorf("GroupAddress(%d) not a group", id)
		}
		if int(a.GroupID()) != id {
			t.Errorf("GroupID = %d, want %d", a.GroupID(), id)
		}
	}
}

func TestParseMAC(t *testing.T) {
	m, err := ParseMAC("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatal(err)
	}
	want := MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if m != want {
		t.Errorf("parsed %v, want %v", m, want)
	}
	if m.GatewayMAC5() != 0xFF {
		t.Errorf("GatewayMAC5 = 0x%02x, want 0xFF", m.GatewayMAC5())
	}
	if m.String() != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("String = %q", m.String())
	}
}

func TestParseMACAcceptsDash(t *testing.T) {
	m, err := ParseMAC("AA-BB-CC-DD-EE-FF")
	if err != nil {
		t.Fatal(err)
	}
	if m[5] != 0xFF {
		t.Errorf("parse with dashes failed: %v", m)
	}
}

func TestParseMACRejectsBadInput(t *testing.T) {
	for _, s := range []string{"", "AA:BB:CC", "AA:BB:CC:DD:EE:FF:GG", "ZZ:BB:CC:DD:EE:FF"} {
		if _, err := ParseMAC(s); err == nil {
			t.Errorf("ParseMAC(%q) expected error", s)
		}
	}
}
