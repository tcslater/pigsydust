package schedule

import "testing"

func TestLocalToUTC_AEST(t *testing.T) {
	// From protocol reference: 8 AM Mon-Fri AEST (UTC+10)
	// → UTC hour 22, repeat 0x4f (Sun-Thu)
	utcHour, utcRepeat := LocalToUTC(8, Weekdays, 10)

	if utcHour != 22 {
		t.Errorf("hour: got %d, want 22", utcHour)
	}
	if utcRepeat != 0x4f {
		t.Errorf("repeat: got 0x%02x, want 0x4f", utcRepeat)
	}
}

func TestLocalToUTC_NoRotation(t *testing.T) {
	// UTC+0: no day boundary crossing.
	utcHour, utcRepeat := LocalToUTC(12, Weekdays, 0)

	if utcHour != 12 {
		t.Errorf("hour: got %d, want 12", utcHour)
	}
	if utcRepeat != Weekdays {
		t.Errorf("repeat: got 0x%02x, want 0x%02x", utcRepeat, Weekdays)
	}
}

func TestLocalToUTC_NegativeOffset(t *testing.T) {
	// 23:00 Mon-Fri in UTC-5 → 04:00 next day, left-rotate
	utcHour, utcRepeat := LocalToUTC(23, Weekdays, -5)

	if utcHour != 4 {
		t.Errorf("hour: got %d, want 4", utcHour)
	}
	// Left-rotate 0x1f by 1: ((0x1f << 1) & 0x7f) | ((0x1f >> 6) & 1)
	// = (0x3e & 0x7f) | 0 = 0x3e
	if utcRepeat != 0x3e {
		t.Errorf("repeat: got 0x%02x, want 0x3e", utcRepeat)
	}
}

func TestLocalToUTC_Daily(t *testing.T) {
	// Daily mask should remain daily regardless of rotation.
	_, utcRepeat := LocalToUTC(2, Daily, 10)

	if utcRepeat != Daily {
		t.Errorf("daily rotation: got 0x%02x, want 0x%02x", utcRepeat, Daily)
	}
}

func TestUTCToLocal_InverseOfLocalToUTC(t *testing.T) {
	tests := []struct {
		name    string
		hour    uint8
		mask    WeekdayMask
		tzOff   int
	}{
		{"AEST", 8, Weekdays, 10},
		{"UTC", 12, Weekdays, 0},
		{"EST", 23, Weekdays, -5},
		{"Daily AEST", 2, Daily, 10},
		{"Sunday only", 6, Sunday, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utcHour, utcRepeat := LocalToUTC(tt.hour, tt.mask, tt.tzOff)
			localHour, localRepeat := UTCToLocal(utcHour, utcRepeat, tt.tzOff)

			if localHour != tt.hour {
				t.Errorf("hour round-trip: got %d, want %d", localHour, tt.hour)
			}
			if localRepeat != tt.mask {
				t.Errorf("repeat round-trip: got 0x%02x, want 0x%02x", localRepeat, tt.mask)
			}
		})
	}
}
