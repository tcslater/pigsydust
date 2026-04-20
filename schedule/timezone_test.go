package schedule

import "testing"

// TestAESTMonFri exercises the worked example from the protocol reference:
// 8:00 AM Mon-Fri in AEST (UTC+10) → hour=22, repeat=0x4f (Sun-Thu UTC).
func TestAESTMonFri(t *testing.T) {
	h, r := LocalToUTC(8, Weekdays, 10)
	if h != 22 {
		t.Errorf("utcHour = %d, want 22", h)
	}
	if r != 0x4F {
		t.Errorf("utcRepeat = 0x%02x, want 0x4F", r)
	}
}

func TestNoRotationSameDay(t *testing.T) {
	h, r := LocalToUTC(15, Weekdays, 10)
	if h != 5 {
		t.Errorf("utcHour = %d, want 5", h)
	}
	if r != Weekdays {
		t.Errorf("utcRepeat = 0x%02x, want 0x1F (unrotated)", r)
	}
}

func TestForwardRotationNegativeOffset(t *testing.T) {
	// Local tz UTC-8 (PST). 6pm local Mon → 2am Tue UTC: day shifts forward.
	h, r := LocalToUTC(18, Monday, -8)
	if h != 2 {
		t.Errorf("utcHour = %d, want 2", h)
	}
	// Monday bit (0x01) should left-rotate to Tuesday bit (0x02).
	if r != Tuesday {
		t.Errorf("utcRepeat = 0x%02x, want 0x02 (Tue)", r)
	}
}

func TestDailyRotationInvariant(t *testing.T) {
	for tz := -12; tz <= 14; tz++ {
		_, r := LocalToUTC(12, Daily, tz)
		if r != Daily {
			t.Errorf("daily rotated at tz=%d: 0x%02x, want 0x7F", tz, r)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		hour   byte
		repeat byte
		tz     int
	}{
		{8, Weekdays, 10},
		{18, Monday, -8},
		{0, Sunday, 10},
		{23, Saturday, -5},
		{12, Daily, 14},
	}
	for _, c := range cases {
		h, r := LocalToUTC(c.hour, c.repeat, c.tz)
		h2, r2 := UTCToLocal(h, r, c.tz)
		if h2 != c.hour || r2 != c.repeat {
			t.Errorf("round-trip h=%d r=0x%02x tz=%d: back to h=%d r=0x%02x",
				c.hour, c.repeat, c.tz, h2, r2)
		}
	}
}
