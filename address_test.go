package piggsydust

import "testing"

func TestGroupAddress(t *testing.T) {
	a := GroupAddress(1)
	if a != 0x8001 {
		t.Errorf("GroupAddress(1): got 0x%04x, want 0x8001", a)
	}

	if !a.IsGroup() {
		t.Error("GroupAddress(1) should be a group")
	}

	id, ok := a.GroupID()
	if !ok || id != 1 {
		t.Errorf("GroupID: got (%d, %v), want (1, true)", id, ok)
	}
}

func TestAddress_IsIndividual(t *testing.T) {
	a := Address(42)
	if !a.IsIndividual() {
		t.Error("Address(42) should be individual")
	}
	if a.IsGroup() {
		t.Error("Address(42) should not be a group")
	}
}

func TestAddress_Broadcast(t *testing.T) {
	if AddressBroadcast.IsGroup() {
		t.Error("broadcast should not be a group")
	}
	if AddressBroadcast.IsIndividual() {
		t.Error("broadcast should not be individual")
	}
}

func TestAddress_String(t *testing.T) {
	tests := []struct {
		addr Address
		want string
	}{
		{AddressBroadcast, "broadcast"},
		{AddressBroadcastPoll, "broadcast-poll"},
		{AddressScheduleCoordinator, "schedule-coordinator"},
		{GroupAddress(3), "group-3"},
		{Address(42), "device-42"},
	}
	for _, tt := range tests {
		if got := tt.addr.String(); got != tt.want {
			t.Errorf("%d.String(): got %q, want %q", tt.addr, got, tt.want)
		}
	}
}
