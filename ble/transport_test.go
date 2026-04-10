package ble

import (
	"testing"

	"github.com/tcslater/piggsydust"
	"tinygo.org/x/bluetooth"
)

// Compile-time interface checks.
var _ piggsydust.Transport = (*Transport)(nil)

func TestUUID_MeshService(t *testing.T) {
	// Verify the mesh service UUID ends with 1910.
	uuid := MeshServiceUUID
	if uuid == (bluetooth.UUID{}) {
		t.Fatal("MeshServiceUUID should not be zero")
	}
}

func TestUUID_Characteristics(t *testing.T) {
	// Verify characteristic UUIDs are distinct and non-zero.
	uuids := []bluetooth.UUID{CharNotifyUUID, CharCmdUUID, CharPairUUID, CharOTAUUID}
	seen := make(map[bluetooth.UUID]bool)

	for _, u := range uuids {
		if u == (bluetooth.UUID{}) {
			t.Error("characteristic UUID should not be zero")
		}
		if seen[u] {
			t.Errorf("duplicate UUID: %s", u.String())
		}
		seen[u] = true
	}
}

func TestServiceUUID(t *testing.T) {
	// The 16-bit service UUID should be 0xCDAB.
	expected := bluetooth.New16BitUUID(0xCDAB)
	if ServiceUUID != expected {
		t.Errorf("ServiceUUID: got %s, want %s", ServiceUUID.String(), expected.String())
	}
}
