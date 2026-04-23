package pigsydust

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// MACAddress is a 6-byte BLE MAC address in standard printed order
// (AA:BB:CC:DD:EE:FF → index 0 = AA, index 5 = FF).
//
// The Pixie / Telink mesh firmware uses the low 4 bytes of the connected
// node's MAC in nonce construction, and the last byte (index 5) is embedded
// as a routing tag in several commands. See the protocol reference.
type MACAddress [6]byte

// String returns the MAC in AA:BB:CC:DD:EE:FF form (uppercase hex).
func (m MACAddress) String() string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		m[0], m[1], m[2], m[3], m[4], m[5])
}

// GatewayMAC5 returns the last byte of the MAC address — the firmware
// "routing tag" cookie required by LED queries, schedule operations, and
// group commands.
func (m MACAddress) GatewayMAC5() byte {
	return m[5]
}

// ParseMAC parses a colon- or dash-separated MAC address.
func ParseMAC(s string) (MACAddress, error) {
	var m MACAddress
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == '-' })
	if len(parts) != 6 {
		return m, fmt.Errorf("pigsydust: MAC %q: want 6 octets, got %d", s, len(parts))
	}
	for i, p := range parts {
		b, err := hex.DecodeString(p)
		if err != nil || len(b) != 1 {
			return m, fmt.Errorf("pigsydust: MAC %q: bad octet %q", s, p)
		}
		m[i] = b[0]
	}
	return m, nil
}
