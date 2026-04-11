package pigsydust

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// MACAddress is a 6-byte Bluetooth MAC address in standard order
// (AA:BB:CC:DD:EE:FF where index 0=AA, index 5=FF).
type MACAddress [6]byte

// String returns the MAC address in colon-separated hex notation.
func (m MACAddress) String() string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		m[0], m[1], m[2], m[3], m[4], m[5])
}

// GatewayMAC5 returns byte index 5 (the last byte) of the MAC address.
// This value is used as a cookie in LED queries, schedule operations,
// and group commands.
func (m MACAddress) GatewayMAC5() byte {
	return m[5]
}

// ParseMAC parses a colon-separated hex MAC address string (e.g. "AA:BB:CC:DD:EE:FF").
func ParseMAC(s string) (MACAddress, error) {
	var m MACAddress
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return m, fmt.Errorf("pigsydust: invalid MAC address %q: expected 6 octets", s)
	}
	for i, p := range parts {
		var b byte
		_, err := fmt.Sscanf(p, "%02x", &b)
		if err != nil {
			return m, fmt.Errorf("pigsydust: invalid MAC address %q: bad octet %q", s, p)
		}
		m[i] = b
	}
	return m, nil
}

// DeviceType indicates the current role of a mesh node.
type DeviceType byte

const (
	// DeviceTypeGateway indicates the node is the current mesh gateway.
	DeviceTypeGateway DeviceType = 0x47

	// DeviceTypeLeaf indicates the node is a regular mesh leaf.
	DeviceTypeLeaf DeviceType = 0x45
)

func (d DeviceType) String() string {
	switch d {
	case DeviceTypeGateway:
		return "gateway"
	case DeviceTypeLeaf:
		return "leaf"
	default:
		return fmt.Sprintf("unknown(0x%02x)", byte(d))
	}
}

// AdvertisementData holds the parsed contents of a Pixie BLE advertisement.
type AdvertisementData struct {
	MeshName   string
	MAC        MACAddress
	DeviceType DeviceType
	NetworkID  uint32
}

// ParseManufacturerData extracts device information from the BLE manufacturer
// data payload. The companyID should be 0x0211 (Skytone). The data parameter
// is the manufacturer data bytes after the 2-byte company ID.
func ParseManufacturerData(companyID uint16, data []byte) (AdvertisementData, error) {
	var ad AdvertisementData

	if companyID != 0x0211 {
		return ad, fmt.Errorf("pigsydust: unexpected manufacturer ID 0x%04x (expected 0x0211)", companyID)
	}

	if len(data) < 21 {
		return ad, fmt.Errorf("pigsydust: manufacturer data too short (%d bytes, need 21)", len(data))
	}

	// MAC bytes at offset 2-5 are in little-endian order: [5,4,3,2]
	ad.MAC[5] = data[2]
	ad.MAC[4] = data[3]
	ad.MAC[3] = data[4]
	ad.MAC[2] = data[5]

	// Device type at offset 14
	ad.DeviceType = DeviceType(data[14])

	// Mesh network ID at offset 17-20 (little-endian 32-bit)
	ad.NetworkID = binary.LittleEndian.Uint32(data[17:21])

	return ad, nil
}
