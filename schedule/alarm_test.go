package schedule

import (
	"testing"

	"github.com/tcslater/pigsydust"
)

func TestCountdownLayout(t *testing.T) {
	r, err := Countdown(0x0042, 5)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID() != CountdownID {
		t.Errorf("id = 0x%02x, want 0xC9", r.ID())
	}
	if r.Kind() != KindCountdown {
		t.Errorf("kind = 0x%02x, want 0x03", byte(r.Kind()))
	}
	if r.Duration() != 5 {
		t.Errorf("duration = %d, want 5", r.Duration())
	}
	if r.Target() != 0x0042 {
		t.Errorf("target = 0x%04x, want 0x0042", uint16(r.Target()))
	}
	if r.Action() != ActionOff {
		t.Errorf("countdown must be OFF-only, got action 0x%02x", byte(r.Action()))
	}
	// state bytes must be zero
	for i := 10; i <= 14; i++ {
		if r[i] != 0 {
			t.Errorf("countdown state byte %d = 0x%02x, must be zero", i, r[i])
		}
	}
}

func TestCountdownZeroDurationRejected(t *testing.T) {
	if _, err := Countdown(0x0001, 0); err == nil {
		t.Error("expected error for zero duration")
	}
}

func TestOneShotStateTail(t *testing.T) {
	r, err := OneShot(0x10, 0x0001, 22, 30, ActionOnFullBright)
	if err != nil {
		t.Fatal(err)
	}
	if r.Repeat() != 0 {
		t.Errorf("one-shot repeat = 0x%02x, want 0", r.Repeat())
	}
	if r.Hour() != 22 || r.Minute() != 30 {
		t.Errorf("time = %d:%d, want 22:30", r.Hour(), r.Minute())
	}
	want := [5]byte{0x00, 0x00, 0xFF, 0xFF, 0xFF}
	for i, w := range want {
		if r[10+i] != w {
			t.Errorf("state byte %d = 0x%02x, want 0x%02x", 10+i, r[10+i], w)
		}
	}
}

func TestRecurringRequiresWeekdayBit(t *testing.T) {
	if _, err := Recurring(0x10, 0x0001, 8, 0, 0, ActionOnFullBright); err == nil {
		t.Error("expected error for repeat=0")
	}
}

func TestRecurringMasksHighBit(t *testing.T) {
	// bit 7 must be stripped — only bits 0-6 are weekdays.
	r, err := Recurring(0x10, 0x0001, 8, 0, 0xFF, ActionOnFullBright)
	if err != nil {
		t.Fatal(err)
	}
	if r.Repeat() != 0x7F {
		t.Errorf("repeat = 0x%02x, want 0x7F (bit 7 stripped)", r.Repeat())
	}
}

func TestXORMatchesFold(t *testing.T) {
	r, err := OneShot(0x11, 0x0005, 10, 15, ActionOnFullBright)
	if err != nil {
		t.Fatal(err)
	}
	var want byte
	for _, b := range r {
		want ^= b
	}
	if got := r.XOR(); got != want {
		t.Errorf("XOR = 0x%02x, want 0x%02x", got, want)
	}
}

func TestSetActiveToggle(t *testing.T) {
	r, err := OneShot(0x11, 0x0005, 10, 15, ActionOnFullBright)
	if err != nil {
		t.Fatal(err)
	}
	off := r.SetActive(false)
	if off.Active() {
		t.Error("SetActive(false) still active")
	}
	if off[7] != 0 {
		t.Errorf("active byte = 0x%02x, want 0", off[7])
	}
	// XOR must differ by exactly bit 0.
	if r.XOR()^off.XOR() != 0x01 {
		t.Errorf("enable toggle XOR delta = 0x%02x, want 0x01",
			r.XOR()^off.XOR())
	}
}

func TestInvalidHourMinute(t *testing.T) {
	if _, err := OneShot(1, 0x0001, 24, 0, ActionOff); err == nil {
		t.Error("expected error for hour=24")
	}
	if _, err := OneShot(1, 0x0001, 0, 60, ActionOff); err == nil {
		t.Error("expected error for minute=60")
	}
}

func TestTransitionKindCheck(t *testing.T) {
	if _, err := Transition(1, 0x0001, 8, 0, Daily, KindRegular, 5, ActionOnFullBright); err == nil {
		t.Error("Transition must reject KindRegular")
	}
	if _, err := Transition(1, 0x0001, 8, 0, Daily, KindFlick, 5, ActionOnFullBright); err != nil {
		t.Errorf("Transition(KindFlick) errored: %v", err)
	}
}

func TestGroupTarget(t *testing.T) {
	r, err := OneShot(1, pigsydust.GroupAddress(5), 8, 0, ActionOnUnchanged)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Target().IsGroup() {
		t.Errorf("target 0x%04x should be a group address", uint16(r.Target()))
	}
	if r.Target().GroupID() != 5 {
		t.Errorf("group id = %d, want 5", r.Target().GroupID())
	}
}
