// Package protocol holds the wire-level constants of the SAL Pixie /
// Telink BLE mesh protocol: opcodes, vendor IDs, characteristic UUIDs,
// and well-known address values.
package protocol

// Telink mesh GATT service and characteristic UUIDs.
const (
	MeshServiceUUID = "00010203-0405-0607-0809-0a0b0c0d1910"
	CharNotifyUUID  = "00010203-0405-0607-0809-0a0b0c0d1911"
	CharCmdUUID     = "00010203-0405-0607-0809-0a0b0c0d1912"
	CharOTAUUID     = "00010203-0405-0607-0809-0a0b0c0d1913"
	CharPairUUID    = "00010203-0405-0607-0809-0a0b0c0d1914"
)

// Device Information Service UUIDs (used for MAC extraction on macOS/iOS).
const (
	DISServiceUUID      = "0000180a-0000-1000-8000-00805f9b34fb"
	DISModelNumberUUID  = "00002a24-0000-1000-8000-00805f9b34fb"
	DISFirmwareRevUUID  = "00002a26-0000-1000-8000-00805f9b34fb"
	DISHardwareRevUUID  = "00002a27-0000-1000-8000-00805f9b34fb"
	DISManufacturerUUID = "00002a29-0000-1000-8000-00805f9b34fb"
)

// BLE advertisement manufacturer ID (Skytone).
const ManufacturerID = 0x0211

// Vendor IDs carried in plaintext command payloads.
const (
	// VendorSkytone is the main application-layer vendor ID (0x6969).
	VendorSkytone = 0x6969
	// VendorSkytoneAlt is used by a handful of opcodes (status poll, group
	// query/probe), value 0x0211 (the manufacturer ID reused as a vendor).
	VendorSkytoneAlt = 0x0211
	// VendorLEDQuery is unique to the 0xd9 LED query opcode, value 0x696b.
	VendorLEDQuery = 0x696B
)

// OpType is the 2-bit operation type in the high bits of the wire opcode
// byte: wire_opcode = (op_type << 6) | (op6 & 0x3f).
const OpTypeClient = 3

// Opcodes — client→device commands.
//
//nolint:revive // constants are named to match the spec.
const (
	OpOnOff         = 0xED // on/off
	OpGroupOnOff    = 0xE7 // group on/off (alternative)
	OpStatusQuery   = 0xC5 // broadcast status query / set_utc
	OpStatusPoll    = 0xDA // unicast or broadcast-poll status poll (keepalive)
	OpSetGroup      = 0xEF // set group membership
	OpQueryGroup    = 0xD7 // query group membership
	OpProbeGroup    = 0xDD // probe group address in use
	OpLEDSet        = 0xFF // LED indicator set
	OpLEDQuery      = 0xD9 // LED indicator query
	OpFindMe        = 0xF5 // find-me (LED flash)
	OpWriteAlarm    = 0xCC // write alarm (2 fragments)
	OpQueryAlarm    = 0xCD // query alarm slot
	OpDeleteAlarm   = 0xCE // delete alarm slot
	OpSlotQuery     = 0xF0 // slot assignment query (after write)
	OpSunriseSunset = 0xD0 // sunrise/sunset schedule (3 fragments)
)

// Opcodes — notifications (device→client).
//
//nolint:revive
const (
	OpNotifyStatusPoll      = 0xDB // 0xDA response (unicast status)
	OpNotifyStatusBroadcast = 0xDC // 0xC5 response / unsolicited state change
	OpNotifyGroupResponse   = 0xD4 // 0xD7 query + 0xDD probe response
	OpNotifyGroupAck        = 0xEE // 0xEF set-membership ack
	OpNotifyLEDOrSlot       = 0xD3 // 0xD9 LED query response or 0xF0 slot response
	OpNotifyAlarmFragment   = 0xC2 // 0xCD alarm-query response fragment
)

// LED set channel select bytes (first byte of each 2-byte channel slot).
const (
	LEDChBlueSelect   = 0xA0
	LEDChOrangeSelect = 0xFF
	// LEDChOrangeInternal is the normalised orange-channel byte firmware
	// reports back in 0xD3 responses (a0/ff are the setter values but
	// firmware stores orange as 0xb6 internally).
	LEDChOrangeInternal = 0xB6
)

// Blue-channel on level (any non-zero lights the blue LED; 0x12 matches
// observed PIXIE app traffic).
const LEDBlueOnLevel = 0x12

// Find-me mode / duration defaults (start = blink for 15s, stop = zeros).
const (
	FindMeModeBlink = 0x03
	FindMeDuration  = 0x0F
)

// ScheduleSlotCount is the total number of alarm slots (0x00 — 0xF9).
const ScheduleSlotCount = 250

// ScheduleSlotEnd is the end-of-list sentinel in 0xCD query responses.
const ScheduleSlotEnd = 0xFF
