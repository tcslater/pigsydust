// Package schedule builds 16-byte alarm records for the Pixie mesh's
// schedule coordinator (address 0x0030).
//
// The record layout and the three alarm kinds — countdown, one-shot, and
// recurring — are documented in the protocol reference. Records are written
// to the mesh via [github.com/tcslater/pigsydust/command.WriteAlarm] as two
// 0xCC fragments, trailed by the XOR fold of all 16 bytes.
//
// All times on the wire are UTC. Use [LocalToUTC] to convert local time and
// rotate the weekday bitmask across day boundaries.
package schedule

import (
	"encoding/binary"
	"fmt"

	"github.com/tcslater/pigsydust"
)

// Kind identifies the alarm kind (record byte 4).
type Kind byte

const (
	KindRegular   Kind = 0x00 // one-shot or recurring
	KindFlick     Kind = 0x01 // flick transition (kind-specific speed)
	KindGradual   Kind = 0x02 // gradual transition (kind-specific speed)
	KindCountdown Kind = 0x03 // fires after duration minutes (OFF-only)
)

// Action is the record byte 15.
type Action byte

const (
	ActionOff           Action = 0x00 // turn off
	ActionOnFullBright  Action = 0x64 // turn on at 100% (device targets)
	ActionOnUnchanged   Action = 0xFF // turn on at current brightness (groups)
)

// Weekday is a bit in the repeat bitmask (bit0 = Monday ... bit6 = Sunday).
//
// The bitmask is UTC-rotated when the local→UTC conversion crosses a day
// boundary — use [LocalToUTC] rather than building it directly for non-UTC
// schedules.
const (
	Monday    byte = 0x01
	Tuesday   byte = 0x02
	Wednesday byte = 0x04
	Thursday  byte = 0x08
	Friday    byte = 0x10
	Saturday  byte = 0x20
	Sunday    byte = 0x40

	Daily      byte = 0x7F
	Weekdays   byte = Monday | Tuesday | Wednesday | Thursday | Friday // 0x1F
	Weekends   byte = Saturday | Sunday                                // 0x60
	NoWeekdays byte = 0x00                                              // one-shot (use OneShot constructor)
)

// CountdownID is the conventional alarm ID for countdown timers (byte 0).
const CountdownID byte = 0xC9

// Record is a 16-byte alarm record.
type Record [16]byte

// ID returns the record's identifier byte (offset 0).
func (r Record) ID() byte { return r[0] }

// Repeat returns the weekday bitmask (offset 1).
func (r Record) Repeat() byte { return r[1] }

// Hour returns the UTC fire hour (offset 2).
func (r Record) Hour() byte { return r[2] }

// Minute returns the UTC fire minute (offset 3).
func (r Record) Minute() byte { return r[3] }

// Kind returns the timer kind (offset 4).
func (r Record) Kind() Kind { return Kind(r[4]) }

// Duration returns the countdown duration in whole minutes (offset 6).
// Meaningful only when Kind() == KindCountdown.
func (r Record) Duration() byte { return r[6] }

// Active reports whether the alarm is enabled (offset 7).
func (r Record) Active() bool { return r[7] != 0 }

// Target returns the device or group address (offset 8-9).
func (r Record) Target() pigsydust.Address {
	return pigsydust.Address(binary.LittleEndian.Uint16(r[8:10]))
}

// Action returns the fire action (offset 15).
func (r Record) Action() Action { return Action(r[15]) }

// XOR returns the XOR-fold of all 16 bytes — the checksum appended to the
// second 0xCC write fragment.
func (r Record) XOR() byte {
	var x byte
	for _, b := range r {
		x ^= b
	}
	return x
}

// SetActive returns a copy of r with the active byte flipped. The enable /
// disable toggle is a plain 0xCC rewrite with only this byte changed; see
// the protocol reference.
func (r Record) SetActive(on bool) Record {
	out := r
	if on {
		out[7] = 0x01
	} else {
		out[7] = 0x00
	}
	return out
}

// Countdown builds an OFF-only countdown timer that fires after durationMin
// whole minutes. Per firmware, all state bytes are zero and the action is
// always ActionOff — a "countdown to ON" must use a regular one-shot instead.
func Countdown(target pigsydust.Address, durationMin byte) (Record, error) {
	if durationMin == 0 {
		return Record{}, fmt.Errorf("schedule: countdown duration must be non-zero")
	}
	var r Record
	r[0] = CountdownID
	r[1] = NoWeekdays
	r[2] = 0 // hour unused
	r[3] = 0 // min unused
	r[4] = byte(KindCountdown)
	r[5] = 0
	r[6] = durationMin
	r[7] = 0x01 // active
	binary.LittleEndian.PutUint16(r[8:10], uint16(target))
	// state bytes 10-14 are zero for countdown
	r[15] = byte(ActionOff)
	return r, nil
}

// OneShot builds a one-shot alarm firing at the given UTC hour:minute.
//
// id must be unique and stable across enable/disable toggles (0xC9 is
// reserved for countdowns). action picks the fire action.
func OneShot(id byte, target pigsydust.Address, utcHour, utcMin byte, action Action) (Record, error) {
	return build(id, 0x00, utcHour, utcMin, KindRegular, 0, target, action)
}

// Recurring builds an alarm firing weekly at the given UTC hour:minute on the
// weekdays set in repeat (bit0 = Monday, bit6 = Sunday).
//
// For non-UTC schedules, use [LocalToUTC] first to get the rotated repeat
// bitmask.
func Recurring(id byte, target pigsydust.Address, utcHour, utcMin byte, repeat byte, action Action) (Record, error) {
	if repeat == 0 {
		return Record{}, fmt.Errorf("schedule: recurring alarm needs at least one weekday bit set — use OneShot instead")
	}
	return build(id, repeat&0x7F, utcHour, utcMin, KindRegular, 0, target, action)
}

// Transition builds a Flick or Gradual alarm with a transition speed byte.
func Transition(id byte, target pigsydust.Address, utcHour, utcMin byte, repeat byte, kind Kind, speed byte, action Action) (Record, error) {
	if kind != KindFlick && kind != KindGradual {
		return Record{}, fmt.Errorf("schedule: Transition requires KindFlick or KindGradual, got 0x%02X", byte(kind))
	}
	return build(id, repeat&0x7F, utcHour, utcMin, kind, speed, target, action)
}

func build(id, repeat, hour, minute byte, kind Kind, speed byte, target pigsydust.Address, action Action) (Record, error) {
	if hour > 23 {
		return Record{}, fmt.Errorf("schedule: hour %d out of range", hour)
	}
	if minute > 59 {
		return Record{}, fmt.Errorf("schedule: minute %d out of range", minute)
	}
	var r Record
	r[0] = id
	r[1] = repeat
	r[2] = hour
	r[3] = minute
	r[4] = byte(kind)
	r[5] = speed
	r[6] = 0
	r[7] = 0x01
	binary.LittleEndian.PutUint16(r[8:10], uint16(target))
	// state tail 00 00 ff ff ff for all non-countdown kinds
	r[10] = 0x00
	r[11] = 0x00
	r[12] = 0xFF
	r[13] = 0xFF
	r[14] = 0xFF
	r[15] = byte(action)
	return r, nil
}
