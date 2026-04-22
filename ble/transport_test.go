package ble

import (
	"testing"

	"github.com/tcslater/pigsydust"
)

// Compile-time interface check.
var _ pigsydust.Transport = (*Transport)(nil)

func TestUUID_MeshService(t *testing.T) {
	if MeshServiceUUID == (UUID{}) {
		t.Fatal("MeshServiceUUID should not be zero")
	}
}

func TestUUID_Characteristics(t *testing.T) {
	uuids := []UUID{CharNotifyUUID, CharCmdUUID, CharPairUUID, CharOTAUUID}
	seen := make(map[UUID]bool)

	for _, u := range uuids {
		if u == (UUID{}) {
			t.Error("characteristic UUID should not be zero")
		}
		if seen[u] {
			t.Errorf("duplicate UUID: %s", u.String())
		}
		seen[u] = true
	}
}

func TestServiceUUID(t *testing.T) {
	expected := new16BitUUID(0xCDAB)
	if ServiceUUID != expected {
		t.Errorf("ServiceUUID: got %s, want %s", ServiceUUID.String(), expected.String())
	}
}
