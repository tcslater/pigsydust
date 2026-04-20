package protocol

// DeviceClass is a canonical Pixie device class identifier.
//
// Values are the post-halving spec lookup keys: key = canonical_type*1000 +
// canonical_stype, where canonical_* = wire_* * 2. See the protocol
// reference, "Device Classes".
//
// Use [DeviceClassLookup] or [DeviceClassName] to resolve raw advertisement
// wire bytes to a canonical class.
type DeviceClass int

// Canonical device classes.
//
// Note: the *2 halving rule has only been wire-confirmed for SWITCH_G2. The
// other entries are computed from the spec table and may be wrong for any
// individual unit. Unknown lookups fall back to a raw-bytes match.
const (
	DeviceClassUnknown DeviceClass = 0

	SGBX0      DeviceClass = 106
	BRIDGE_G2  DeviceClass = 2004
	POL        DeviceClass = 2014
	BRIDGE     DeviceClass = 2022
	SGB        DeviceClass = 2028
	ACF_DUCTED DeviceClass = 2102
	SPO3       DeviceClass = 4016
	ACF_VRV    DeviceClass = 4102
	SGBX       DeviceClass = 4104
	SGBX2      DeviceClass = 4106
	VFAN_ONLY  DeviceClass = 10030
	FAN_ONLY   DeviceClass = 12030
	BFAN_ONLY  DeviceClass = 14096
	ZCL        DeviceClass = 16108
	FAN_ONLY9  DeviceClass = 18030
	DRC        DeviceClass = 20004
	BSC        DeviceClass = 22004
	GDC1       DeviceClass = 24002
	GDC1_SW    DeviceClass = 24004
	GDC1_SL    DeviceClass = 24006
	GDC1_W     DeviceClass = 24016
	GDC2       DeviceClass = 24034
	GDC1_M2    DeviceClass = 26002
	GDC1_M2W   DeviceClass = 26004
	GDC1_M2L   DeviceClass = 26006
	DV02       DeviceClass = 34048
	RFD        DeviceClass = 40026
	RFD2_SCAN  DeviceClass = 40104
	TSWITCH    DeviceClass = 42024
	TSWITCHG2  DeviceClass = 42026
	ECL_AC     DeviceClass = 42096
	SWITCH     DeviceClass = 44002
	SWITCH_G2  DeviceClass = 44024
	SWITCH_G3  DeviceClass = 44026
	DIMMER     DeviceClass = 46022
	DIMMER_G2  DeviceClass = 46024
	DIMMER_G3  DeviceClass = 46026
	STRIP_W    DeviceClass = 48004
	FCS        DeviceClass = 48006
	SFI_8266   DeviceClass = 48064
	SFI_825X   DeviceClass = 48066
	DM10       DeviceClass = 48096
	DALI_DT6   DeviceClass = 48098
	RCT_W      DeviceClass = 48100
	RFD2       DeviceClass = 48104
	STRIP2_CCT DeviceClass = 50008
	RFD_CT     DeviceClass = 50026
	RCT_CCT    DeviceClass = 50100
	RFD2_CT    DeviceClass = 50104
	STRIP2_RGBCCT DeviceClass = 52008
	RCT_RGBW   DeviceClass = 52100
	STRIP_RGB  DeviceClass = 54004
	FCR        DeviceClass = 54006
	STRIP2_RGB DeviceClass = 54008
	RCT_RGB    DeviceClass = 54100
	RGB_X      DeviceClass = 54198
	RCT_RGBCCT DeviceClass = 56100
	IR36       DeviceClass = 60002
	IR12       DeviceClass = 60004
	SMR        DeviceClass = 60006
	DRS        DeviceClass = 60020
	DRSM2      DeviceClass = 60022
	DRSM3      DeviceClass = 60024
	CAP        DeviceClass = 102002
	MTW        DeviceClass = 102004
	STC        DeviceClass = 102006
	MTW2_AL    DeviceClass = 102008
	MTW2_AN    DeviceClass = 102010
	MRC        DeviceClass = 102012
	CAP3       DeviceClass = 102014
	SGB3       DeviceClass = 102016
	SIC        DeviceClass = 102020
	DIAL       DeviceClass = 102040
	VFAN_CT    DeviceClass = 106030
	FAN_CT     DeviceClass = 108030
	FAN_CT9    DeviceClass = 114030
	SONOS      DeviceClass = 180002

	// ACF_RS8 matches wire stype == 0x39 (canonical stype 114) for any
	// type. Synthetic key outside the normal type*1000+stype space so
	// direct lookups don't collide.
	ACF_RS8 DeviceClass = 10000114
)

var deviceClassNames = map[DeviceClass]string{
	SGBX0:         "SGBX0",
	BRIDGE_G2:     "BRIDGE_G2",
	POL:           "POL",
	BRIDGE:        "BRIDGE",
	SGB:           "SGB",
	ACF_DUCTED:    "ACF_DUCTED",
	SPO3:          "SPO3",
	ACF_VRV:       "ACF_VRV",
	SGBX:          "SGBX",
	SGBX2:         "SGBX2",
	VFAN_ONLY:     "VFAN_ONLY",
	FAN_ONLY:      "FAN_ONLY",
	BFAN_ONLY:     "BFAN_ONLY",
	ZCL:           "ZCL",
	FAN_ONLY9:     "FAN_ONLY9",
	DRC:           "DRC",
	BSC:           "BSC",
	GDC1:          "GDC1",
	GDC1_SW:       "GDC1_SW",
	GDC1_SL:       "GDC1_SL",
	GDC1_W:        "GDC1_W",
	GDC2:          "GDC2",
	GDC1_M2:       "GDC1_M2",
	GDC1_M2W:      "GDC1_M2W",
	GDC1_M2L:      "GDC1_M2L",
	DV02:          "DV02",
	RFD:           "RFD",
	RFD2_SCAN:     "RFD2_SCAN",
	TSWITCH:       "TSWITCH",
	TSWITCHG2:     "TSWITCHG2",
	ECL_AC:        "ECL_AC",
	SWITCH:        "SWITCH",
	SWITCH_G2:     "SWITCH_G2",
	SWITCH_G3:     "SWITCH_G3",
	DIMMER:        "DIMMER",
	DIMMER_G2:     "DIMMER_G2",
	DIMMER_G3:     "DIMMER_G3",
	STRIP_W:       "STRIP_W",
	FCS:           "FCS",
	SFI_8266:      "SFI_8266",
	SFI_825X:      "SFI_825X",
	DM10:          "DM10",
	DALI_DT6:      "DALI_DT6",
	RCT_W:         "RCT_W",
	RFD2:          "RFD2",
	STRIP2_CCT:    "STRIP2_CCT",
	RFD_CT:        "RFD_CT",
	RCT_CCT:       "RCT_CCT",
	RFD2_CT:       "RFD2_CT",
	STRIP2_RGBCCT: "STRIP2_RGBCCT",
	RCT_RGBW:      "RCT_RGBW",
	STRIP_RGB:     "STRIP_RGB",
	FCR:           "FCR",
	STRIP2_RGB:    "STRIP2_RGB",
	RCT_RGB:       "RCT_RGB",
	RGB_X:         "RGB_X",
	RCT_RGBCCT:    "RCT_RGBCCT",
	IR36:          "IR36",
	IR12:          "IR12",
	SMR:           "SMR",
	DRS:           "DRS",
	DRSM2:         "DRSM2",
	DRSM3:         "DRSM3",
	CAP:           "CAP",
	MTW:           "MTW",
	STC:           "STC",
	MTW2_AL:       "MTW2_AL",
	MTW2_AN:       "MTW2_AN",
	MRC:           "MRC",
	CAP3:          "CAP3",
	SGB3:          "SGB3",
	SIC:           "SIC",
	DIAL:          "DIAL",
	VFAN_CT:       "VFAN_CT",
	FAN_CT:        "FAN_CT",
	FAN_CT9:       "FAN_CT9",
	SONOS:         "SONOS",
	ACF_RS8:       "ACF_RS8",
}

var deviceClassAliases = map[DeviceClass]DeviceClass{
	// Wire (22, 11) → canonical (44, 22) — alternate form reported for SWITCH.
	44022: SWITCH,
}

// DeviceClassLookup resolves advertisement wire bytes to a canonical
// [DeviceClass].
//
// It applies the *2 halving rule, the ACF_RS8 shortcut (wire stype 0x39),
// the alias table, and finally a raw-bytes fallback. Returns
// [DeviceClassUnknown] for values not in the table.
func DeviceClassLookup(wireType, wireStype byte) DeviceClass {
	if wireStype == 0x39 {
		return ACF_RS8
	}
	canonical := DeviceClass(int(wireType)*2*1000 + int(wireStype)*2)
	if _, ok := deviceClassNames[canonical]; ok {
		return canonical
	}
	if alias, ok := deviceClassAliases[canonical]; ok {
		return alias
	}
	raw := DeviceClass(int(wireType)*1000 + int(wireStype))
	if _, ok := deviceClassNames[raw]; ok {
		return raw
	}
	if alias, ok := deviceClassAliases[raw]; ok {
		return alias
	}
	return DeviceClassUnknown
}

// DeviceClassName returns the canonical name string for a device class, or
// the empty string if unknown.
func DeviceClassName(wireType, wireStype byte) string {
	return deviceClassNames[DeviceClassLookup(wireType, wireStype)]
}

// Name returns the symbolic name for the class, or "" if unknown.
func (c DeviceClass) Name() string {
	return deviceClassNames[c]
}
