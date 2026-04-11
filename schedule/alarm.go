// Package schedule provides types and utilities for building and parsing
// Telink BLE mesh alarm (schedule/timer) records.
package schedule

import (
	"fmt"

	"github.com/tcslater/pigsydust/internal/byteutil"
)

// AlarmType distinguishes regular timers from countdown timers.
type AlarmType uint16

const (
	// AlarmRegular is a regular (one-shot or recurring) alarm.
	AlarmRegular AlarmType = 0x0000

	// AlarmCountdown is a countdown timer that fires after a duration.
	AlarmCountdown AlarmType = 0x0003
)

// Action specifies what happens when an alarm fires.
type Action byte

const (
	// ActionOff turns the target off.
	ActionOff Action = 0x00

	// ActionOnFull turns the target on at 100% brightness.
	ActionOnFull Action = 0x64

	// ActionOnCurrent turns the target on at its current/unchanged brightness.
	ActionOnCurrent Action = 0xff
)

// WeekdayMask is a 7-bit bitmask for recurring alarm days.
// bit0=Monday through bit6=Sunday.
type WeekdayMask uint8

const (
	Monday    WeekdayMask = 1 << 0
	Tuesday   WeekdayMask = 1 << 1
	Wednesday WeekdayMask = 1 << 2
	Thursday  WeekdayMask = 1 << 3
	Friday    WeekdayMask = 1 << 4
	Saturday  WeekdayMask = 1 << 5
	Sunday    WeekdayMask = 1 << 6
	Weekdays  WeekdayMask = 0x1f // Monday through Friday
	Daily     WeekdayMask = 0x7f // All days
)

// AlarmRecord represents a 16-byte alarm record stored on the mesh coordinator.
type AlarmRecord struct {
	// ID is a monotonic alarm identifier. Use 0xc9 for countdowns.
	// IDs must be consistent across enable/disable toggles of the same alarm.
	ID byte

	// Repeat is the weekday bitmask in UTC. Use [LocalToUTC] to convert
	// from local time. 0x00 = one-shot.
	Repeat WeekdayMask

	// Hour is the fire hour in UTC (0-23).
	Hour uint8

	// Minute is the fire minute in UTC (0-59).
	Minute uint8

	// Type distinguishes regular alarms from countdowns.
	Type AlarmType

	// Duration is the countdown duration in whole minutes.
	// Only used when Type is AlarmCountdown.
	Duration uint8

	// Active indicates whether the alarm is enabled.
	Active bool

	// Target is the device or group address to act on.
	// Groups use 0x8000 | id.
	Target uint16

	// Action specifies the on/off/brightness action when the alarm fires.
	Action Action
}

// MarshalBinary serializes the alarm record to the 16-byte wire format.
func (a AlarmRecord) MarshalBinary() ([]byte, error) {
	b := make([]byte, 16)

	b[0] = a.ID
	b[1] = byte(a.Repeat)
	b[2] = a.Hour
	b[3] = a.Minute
	byteutil.PutLE16(b[4:6], uint16(a.Type))
	b[6] = a.Duration

	if a.Active {
		b[7] = 0x01
	}

	byteutil.PutLE16(b[8:10], a.Target)

	// State bytes (offset 10-14) depend on alarm type and action.
	if a.Type == AlarmCountdown {
		// Countdown: state bytes all zero (firmware-enforced).
	} else {
		// Regular: 00 00 ff ff ff
		b[10] = 0x00
		b[11] = 0x00
		b[12] = 0xff
		b[13] = 0xff
		b[14] = 0xff
	}

	b[15] = byte(a.Action)

	return b, nil
}

// UnmarshalBinary deserializes a 16-byte wire format into an alarm record.
func (a *AlarmRecord) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("pigsydust/schedule: alarm record too short (%d bytes, need 16)", len(data))
	}

	a.ID = data[0]
	a.Repeat = WeekdayMask(data[1])
	a.Hour = data[2]
	a.Minute = data[3]
	a.Type = AlarmType(byteutil.LE16(data[4:6]))
	a.Duration = data[6]
	a.Active = data[7] != 0
	a.Target = byteutil.LE16(data[8:10])
	a.Action = Action(data[15])

	return nil
}

// XORChecksum computes the XOR-fold of the 16-byte wire representation.
func (a AlarmRecord) XORChecksum() byte {
	data, _ := a.MarshalBinary()
	return byteutil.XORFold(data)
}

// StoredAlarm pairs an alarm record with its coordinator slot index.
type StoredAlarm struct {
	Slot  uint8
	Alarm AlarmRecord
}

// NewCountdown creates a countdown timer alarm record.
// Countdown timers are OFF-only by firmware design.
func NewCountdown(durationMinutes uint8, target uint16) AlarmRecord {
	return AlarmRecord{
		ID:       0xc9,
		Type:     AlarmCountdown,
		Duration: durationMinutes,
		Active:   true,
		Target:   target,
		Action:   ActionOff,
	}
}

// NewOneShotAlarm creates a one-shot alarm at the given UTC hour and minute.
func NewOneShotAlarm(id byte, utcHour, utcMinute uint8, target uint16, action Action) AlarmRecord {
	return AlarmRecord{
		ID:     id,
		Hour:   utcHour,
		Minute: utcMinute,
		Type:   AlarmRegular,
		Active: true,
		Target: target,
		Action: action,
	}
}

// NewRecurringAlarm creates a recurring alarm. The localHour and days are
// automatically converted to UTC using the provided timezone offset.
func NewRecurringAlarm(id byte, localHour, localMinute uint8, days WeekdayMask, tzOffsetHours int, target uint16, action Action) AlarmRecord {
	utcHour, utcDays := LocalToUTC(localHour, days, tzOffsetHours)

	return AlarmRecord{
		ID:     id,
		Repeat: utcDays,
		Hour:   utcHour,
		Minute: localMinute,
		Type:   AlarmRegular,
		Active: true,
		Target: target,
		Action: action,
	}
}
