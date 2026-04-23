package schedule

// LocalToUTC converts a local fire hour and weekday repeat bitmask to the UTC
// hour and UTC-rotated bitmask that belong on the wire.
//
// When the UTC conversion crosses a day boundary backward (local - tz < 0),
// the 7-bit repeat mask is right-rotated by 1. When it crosses forward
// (local - tz >= 24), the mask is left-rotated by 1. The rotation preserves
// the daily / weekdays / weekends patterns across the TZ shift.
//
// tzOffsetHours is the local timezone's offset from UTC, e.g. +10 for AEST.
// localHour must be in [0, 23]. Minutes do not shift the day, so are passed
// through unchanged and not handled here.
func LocalToUTC(localHour byte, repeat byte, tzOffsetHours int) (utcHour byte, utcRepeat byte) {
	adjusted := int(localHour) - tzOffsetHours + 24
	utcHour = byte(((int(localHour) - tzOffsetHours) + 24*2) % 24)
	r := repeat & 0x7F
	switch {
	case adjusted < 24:
		utcRepeat = ((r >> 1) | ((r & 0x01) << 6)) & 0x7F
	case adjusted >= 48:
		utcRepeat = ((r << 1) & 0x7F) | ((r >> 6) & 0x01)
	default:
		utcRepeat = r
	}
	return utcHour, utcRepeat
}

// UTCToLocal is the inverse of [LocalToUTC]. A record read back from the
// coordinator is always UTC; callers wanting to display it in local time
// should apply this conversion with their current tz offset.
func UTCToLocal(utcHour byte, utcRepeat byte, tzOffsetHours int) (localHour byte, localRepeat byte) {
	adjusted := int(utcHour) + tzOffsetHours + 24
	localHour = byte(((int(utcHour) + tzOffsetHours) + 24*2) % 24)
	r := utcRepeat & 0x7F
	switch {
	case adjusted < 24:
		localRepeat = ((r >> 1) | ((r & 0x01) << 6)) & 0x7F
	case adjusted >= 48:
		localRepeat = ((r << 1) & 0x7F) | ((r >> 6) & 0x01)
	default:
		localRepeat = r
	}
	return localHour, localRepeat
}
