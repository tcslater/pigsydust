package schedule

// LocalToUTC converts a local hour and weekday mask to UTC, handling
// day-boundary crossings with 7-bit bitmask rotation.
//
// When the UTC conversion crosses a day boundary:
//   - Day goes backward (adjusted < 24): right-rotate the mask by 1
//   - Day goes forward (adjusted >= 48): left-rotate the mask by 1
//
// Example: "8:00 AM Mon-Fri" in AEST (UTC+10):
//
//	localHour=8, repeat=0x1f, tz=10
//	utcHour = (8 - 10 + 24) % 24 = 22
//	adjusted = 22 < 24 → right-rotate: 0x1f → 0x4f
//	Result: hour=22, repeat=0x4f (Sun-Thu UTC = Mon-Fri AEST)
func LocalToUTC(localHour uint8, repeat WeekdayMask, tzOffsetHours int) (utcHour uint8, utcRepeat WeekdayMask) {
	adjusted := int(localHour) - tzOffsetHours + 24
	utcHour = uint8(adjusted % 24)

	r := repeat
	if adjusted < 24 {
		// Day went backward: right-rotate 7-bit mask by 1.
		r = (r >> 1) | ((r & 1) << 6)
	} else if adjusted >= 48 {
		// Day went forward: left-rotate 7-bit mask by 1.
		r = ((r << 1) & 0x7f) | ((r >> 6) & 1)
	}

	utcRepeat = r & 0x7f
	return utcHour, utcRepeat
}

// UTCToLocal converts a UTC hour and weekday mask to local time,
// applying the inverse rotation of [LocalToUTC].
func UTCToLocal(utcHour uint8, repeat WeekdayMask, tzOffsetHours int) (localHour uint8, localRepeat WeekdayMask) {
	adjusted := int(utcHour) + tzOffsetHours + 24
	localHour = uint8(adjusted % 24)

	r := repeat
	if adjusted < 24 {
		// Day went backward: right-rotate 7-bit mask by 1.
		r = (r >> 1) | ((r & 1) << 6)
	} else if adjusted >= 48 {
		// Day went forward: left-rotate 7-bit mask by 1.
		r = ((r << 1) & 0x7f) | ((r >> 6) & 1)
	}

	localRepeat = r & 0x7f
	return localHour, localRepeat
}
