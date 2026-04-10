package schedule

import "testing"

func TestAlarmRecord_MarshalUnmarshal(t *testing.T) {
	alarm := AlarmRecord{
		ID:     0x01,
		Repeat: Weekdays,
		Hour:   22,
		Minute: 30,
		Type:   AlarmRegular,
		Active: true,
		Target: 0xFFFF,
		Action: ActionOnFull,
	}

	data, err := alarm.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 16 {
		t.Fatalf("length: got %d, want 16", len(data))
	}

	var decoded AlarmRecord
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != alarm.ID {
		t.Errorf("ID: got %d, want %d", decoded.ID, alarm.ID)
	}
	if decoded.Repeat != alarm.Repeat {
		t.Errorf("Repeat: got 0x%02x, want 0x%02x", decoded.Repeat, alarm.Repeat)
	}
	if decoded.Hour != alarm.Hour {
		t.Errorf("Hour: got %d, want %d", decoded.Hour, alarm.Hour)
	}
	if decoded.Minute != alarm.Minute {
		t.Errorf("Minute: got %d, want %d", decoded.Minute, alarm.Minute)
	}
	if decoded.Type != alarm.Type {
		t.Errorf("Type: got %d, want %d", decoded.Type, alarm.Type)
	}
	if decoded.Active != alarm.Active {
		t.Errorf("Active: got %v, want %v", decoded.Active, alarm.Active)
	}
	if decoded.Target != alarm.Target {
		t.Errorf("Target: got 0x%04x, want 0x%04x", decoded.Target, alarm.Target)
	}
	if decoded.Action != alarm.Action {
		t.Errorf("Action: got 0x%02x, want 0x%02x", decoded.Action, alarm.Action)
	}
}

func TestAlarmRecord_StateBytes(t *testing.T) {
	// Regular alarm should have state bytes: 00 00 ff ff ff
	alarm := AlarmRecord{
		Type:   AlarmRegular,
		Active: true,
		Target: 0x0001,
		Action: ActionOnFull,
	}

	data, _ := alarm.MarshalBinary()

	if data[10] != 0x00 || data[11] != 0x00 {
		t.Errorf("state[0:2]: got %02x %02x, want 00 00", data[10], data[11])
	}
	if data[12] != 0xff || data[13] != 0xff || data[14] != 0xff {
		t.Errorf("state[2:5]: got %02x %02x %02x, want ff ff ff", data[12], data[13], data[14])
	}

	// Countdown should have all-zero state bytes.
	countdown := AlarmRecord{
		Type:   AlarmCountdown,
		Active: true,
		Target: 0x0001,
		Action: ActionOff,
	}

	data, _ = countdown.MarshalBinary()
	for i := 10; i < 15; i++ {
		if data[i] != 0x00 {
			t.Errorf("countdown state[%d]: got 0x%02x, want 0x00", i, data[i])
		}
	}
}

func TestAlarmRecord_XORChecksum(t *testing.T) {
	alarm := AlarmRecord{
		ID:     0x01,
		Repeat: Weekdays,
		Hour:   8,
		Minute: 0,
		Active: true,
		Target: 0xFFFF,
		Action: ActionOnFull,
	}

	data, _ := alarm.MarshalBinary()

	// Manually compute XOR.
	var expected byte
	for _, b := range data {
		expected ^= b
	}

	if alarm.XORChecksum() != expected {
		t.Errorf("checksum: got 0x%02x, want 0x%02x", alarm.XORChecksum(), expected)
	}
}

func TestNewCountdown(t *testing.T) {
	alarm := NewCountdown(30, 0x0001)

	if alarm.ID != 0xc9 {
		t.Errorf("ID: got 0x%02x, want 0xc9", alarm.ID)
	}
	if alarm.Type != AlarmCountdown {
		t.Errorf("Type: got %d, want %d", alarm.Type, AlarmCountdown)
	}
	if alarm.Duration != 30 {
		t.Errorf("Duration: got %d, want 30", alarm.Duration)
	}
	if alarm.Action != ActionOff {
		t.Errorf("Action: got 0x%02x, want 0x00 (off-only)", alarm.Action)
	}
}

func TestNewRecurringAlarm(t *testing.T) {
	// 8 AM Mon-Fri in AEST (UTC+10) → 22:00 Sun-Thu UTC
	alarm := NewRecurringAlarm(0x01, 8, 0, Weekdays, 10,
		0xFFFF, ActionOnFull)

	if alarm.Hour != 22 {
		t.Errorf("Hour: got %d, want 22", alarm.Hour)
	}
	if alarm.Repeat != 0x4f {
		t.Errorf("Repeat: got 0x%02x, want 0x4f (Sun-Thu)", alarm.Repeat)
	}
}
