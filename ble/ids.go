package ble

// UUID is a 128-bit Bluetooth UUID in big-endian byte order (most-significant
// byte first), matching the common wire/string representation. It is
// platform-neutral so the same constants work across the Darwin/tinygo and
// Linux/BlueZ code paths; each platform converts to its own native type at
// the call boundary.
type UUID [16]byte

// String returns the canonical 8-4-4-4-12 hexadecimal form.
func (u UUID) String() string {
	const hex = "0123456789abcdef"
	var buf [36]byte
	j := 0
	for i, b := range u {
		switch i {
		case 4, 6, 8, 10:
			buf[j] = '-'
			j++
		}
		buf[j] = hex[b>>4]
		buf[j+1] = hex[b&0x0f]
		j += 2
	}
	return string(buf[:])
}

// new16BitUUID expands a 16-bit UUID into the Bluetooth Base UUID
// (0000xxxx-0000-1000-8000-00805F9B34FB).
func new16BitUUID(v uint16) UUID {
	return UUID{
		0x00, 0x00, byte(v >> 8), byte(v),
		0x00, 0x00, 0x10, 0x00,
		0x80, 0x00, 0x00, 0x80,
		0x5f, 0x9b, 0x34, 0xfb,
	}
}

// ServiceUUID16 is the 16-bit Pixie advertisement service UUID (0xCDAB).
const ServiceUUID16 uint16 = 0xCDAB

// Well-known UUIDs for the Telink mesh GATT service and characteristics.
var (
	// ServiceUUID is the 128-bit form of the Pixie advertisement service UUID.
	ServiceUUID = new16BitUUID(ServiceUUID16)

	// MeshServiceUUID is the primary Telink mesh GATT service.
	MeshServiceUUID = UUID{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x10,
	}

	// CharNotifyUUID is CHAR_NOTIFY — subscribe for encrypted notifications.
	CharNotifyUUID = UUID{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x11,
	}

	// CharCmdUUID is CHAR_CMD — write encrypted command packets.
	CharCmdUUID = UUID{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x12,
	}

	// CharOTAUUID is CHAR_OTA — OTA/config (not used in normal operation).
	CharOTAUUID = UUID{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x13,
	}

	// CharPairUUID is CHAR_PAIR — login handshake and heartbeat.
	CharPairUUID = UUID{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x19, 0x14,
	}

	// disServiceUUID is Device Information Service.
	disServiceUUID = new16BitUUID(0x180a)
	// disModelNumberUUID is the Model Number String characteristic.
	disModelNumberUUID = new16BitUUID(0x2a24)
)

// ManufacturerIDSkytone is the BLE manufacturer company ID for Pixie devices.
const ManufacturerIDSkytone = 0x0211
