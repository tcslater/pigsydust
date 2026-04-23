package pigsydust

import (
	"encoding/binary"
	"fmt"

	"github.com/tcslater/pigsydust/protocol"
)

// Advertisement is a parsed Pixie manufacturer-data advertisement.
type Advertisement struct {
	// MeshName is the advertised BLE local name (the user-visible mesh
	// identifier). Populated by the scanner — ParseAdvertisement doesn't
	// know it because it only sees manufacturer data.
	MeshName string
	// MAC is the full 6-byte hardware MAC of the advertising node. The
	// lower four bytes come from the manufacturer data; the upper two are
	// the fixed Telink/Skytone OUI 00:21.
	MAC           MACAddress
	DeviceType    byte // advert byte 6 — wire-halved device class type
	DeviceSubtype byte // advert byte 7 — wire-halved device class subtype
	StatusByte    byte // advert byte 8 — packed online/alarmDev/version
	StatusFlags   StatusFlags
	MeshAddress   byte   // advert byte 9 — low byte of mesh address (= MAC[5])
	NetworkID     uint32 // advert bytes 11-14, little-endian
	Raw           []byte
}

// AdvertisementData is an alias for [Advertisement], kept for API
// compatibility with external transport modules (e.g. the ble/ module).
type AdvertisementData = Advertisement

// DeviceTypeGateway is the advertised wire device-type byte observed on
// nodes acting as the current mesh gateway. It's a heuristic — the protocol
// reference doesn't define a canonical "gateway" class, but scan filters
// historically use this value to prefer gateway-capable nodes.
const DeviceTypeGateway byte = 0x47

// ParseManufacturerData extracts advertisement fields from the raw
// manufacturer-data payload for the given company ID. companyID must be
// [protocol.ManufacturerID] (0x0211); any other value is rejected.
func ParseManufacturerData(companyID uint16, data []byte) (Advertisement, error) {
	if companyID != protocol.ManufacturerID {
		return Advertisement{}, fmt.Errorf("pigsydust: unexpected manufacturer ID 0x%04X (expected 0x0211)", companyID)
	}
	adv := ParseAdvertisement(data)
	if adv == nil {
		return Advertisement{}, fmt.Errorf("pigsydust: manufacturer data too short (%d bytes, need >= 15)", len(data))
	}
	return *adv, nil
}

// DeviceClass resolves the wire bytes to a canonical device class.
func (a Advertisement) DeviceClass() protocol.DeviceClass {
	return protocol.DeviceClassLookup(a.DeviceType, a.DeviceSubtype)
}

// ParseAdvertisement parses the manufacturer-data buffer for
// [protocol.ManufacturerID] (0x0211). Returns nil if the buffer is too
// short to hold the network identifier (15 bytes minimum).
//
// Some BLE stacks prepend the company ID to the buffer; others don't.
// Pixie firmware always re-emits the company ID at offset 0-1, so we
// accept the buffer as-is and index from the start of what the host BLE
// stack hands us.
func ParseAdvertisement(data []byte) *Advertisement {
	if len(data) < 15 {
		return nil
	}
	var mac MACAddress
	mac[0], mac[1] = 0x00, 0x21 // fixed Telink/Skytone OUI
	mac[5] = data[2]
	mac[4] = data[3]
	mac[3] = data[4]
	mac[2] = data[5]

	ad := &Advertisement{
		MAC:           mac,
		DeviceType:    data[6],
		DeviceSubtype: data[7],
		StatusByte:    data[8],
		StatusFlags:   ParseStatusFlags(data[8]),
		MeshAddress:   data[9],
		NetworkID:     binary.LittleEndian.Uint32(data[11:15]),
		Raw:           append([]byte(nil), data...),
	}
	return ad
}
