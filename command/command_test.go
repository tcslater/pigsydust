package command

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/tcslater/pigsydust/protocol"
)

func wireOpcode(op byte) byte {
	return (protocol.OpTypeClient << 6) | (op & 0x3F)
}

func TestOnOffEncode(t *testing.T) {
	buf := OnOff(0x0001, true).Encode()
	if len(buf) != 15 {
		t.Fatalf("len = %d, want 15", len(buf))
	}
	if dst := binary.LittleEndian.Uint16(buf[0:2]); dst != 0x0001 {
		t.Errorf("dst = 0x%04x, want 0x0001", dst)
	}
	if buf[2] != wireOpcode(protocol.OpOnOff) {
		t.Errorf("opcode byte = 0x%02x, want 0x%02x", buf[2], wireOpcode(protocol.OpOnOff))
	}
	if v := binary.LittleEndian.Uint16(buf[3:5]); v != protocol.VendorSkytone {
		t.Errorf("vendor = 0x%04x, want 0x%04x", v, protocol.VendorSkytone)
	}
	if buf[5] != 0x01 {
		t.Errorf("state byte = 0x%02x, want 0x01", buf[5])
	}
}

func TestOnOffOff(t *testing.T) {
	buf := OnOff(0x0003, false).Encode()
	if buf[5] != 0x00 {
		t.Errorf("state byte = 0x%02x, want 0x00", buf[5])
	}
}

func TestOnOffBroadcast(t *testing.T) {
	buf := OnOff(0xFFFF, true).Encode()
	if dst := binary.LittleEndian.Uint16(buf[0:2]); dst != 0xFFFF {
		t.Errorf("broadcast dst = 0x%04x, want 0xFFFF", dst)
	}
}

func TestStatusQueryEncode(t *testing.T) {
	buf := StatusQuery().Encode()
	if len(buf) != 10 {
		t.Fatalf("len = %d, want 10", len(buf))
	}
	if dst := binary.LittleEndian.Uint16(buf[0:2]); dst != 0xFFFF {
		t.Errorf("dst = 0x%04x, want broadcast", dst)
	}
	if buf[2] != wireOpcode(protocol.OpStatusQuery) {
		t.Errorf("opcode byte = 0x%02x, want 0x%02x", buf[2], wireOpcode(protocol.OpStatusQuery))
	}
}

func TestStatusPollEncode(t *testing.T) {
	buf := StatusPoll(0x007F).Encode()
	if len(buf) != 7 {
		t.Fatalf("len = %d, want 7", len(buf))
	}
	if v := binary.LittleEndian.Uint16(buf[3:5]); v != protocol.VendorSkytoneAlt {
		t.Errorf("status poll uses vendor 0x%04x, want 0x0211", v)
	}
}

func TestSetUTCEncode(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	buf := SetUTC(ts).Encode()
	if len(buf) != 15 {
		t.Fatalf("len = %d, want 15", len(buf))
	}
	if got := binary.LittleEndian.Uint32(buf[5:9]); got != 1700000000 {
		t.Errorf("tv_sec = %d, want 1700000000", got)
	}
	if buf[9] != 0x00 {
		t.Errorf("tz byte = 0x%02x, must be 0x00 (firmware offsets clock by it)", buf[9])
	}
}

func TestSetGroupMembershipEncode(t *testing.T) {
	buf := SetGroupMembership(0x007D, []byte{0x02}, 0xEE).Encode()
	if len(buf) != 15 {
		t.Fatalf("len = %d, want 15", len(buf))
	}
	if buf[2] != wireOpcode(protocol.OpSetGroup) {
		t.Errorf("opcode byte = 0x%02x", buf[2])
	}
	if buf[5] != 0x01 {
		t.Errorf("group count = %d, want 1", buf[5])
	}
	if buf[6] != 0xEE {
		t.Errorf("gw_mac5 = 0x%02x, want 0xEE", buf[6])
	}
	if buf[7] != 0x02 {
		t.Errorf("grp_low[0] = 0x%02x, want 0x02", buf[7])
	}
}

func TestQueryGroupMembershipVendor(t *testing.T) {
	buf := QueryGroupMembership(0x007D).Encode()
	if v := binary.LittleEndian.Uint16(buf[3:5]); v != protocol.VendorSkytoneAlt {
		t.Errorf("vendor = 0x%04x, want 0x0211", v)
	}
}

func TestLEDSetBlueOnlyBlueBytes(t *testing.T) {
	buf := LEDSetBlue(0x007D, true).Encode()
	if buf[5] != protocol.LEDChBlueSelect {
		t.Errorf("blue ch select = 0x%02x, want 0xA0", buf[5])
	}
	if buf[6] == 0 {
		t.Error("blue level must be non-zero for 'on'")
	}
	if buf[7] != 0 || buf[8] != 0 {
		t.Errorf("orange bytes must be zero when only touching blue, got %02x %02x", buf[7], buf[8])
	}
}

func TestLEDSetOrangeLevelMasked(t *testing.T) {
	buf := LEDSetOrange(0x007D, 0xFF).Encode()
	if buf[5] != 0 || buf[6] != 0 {
		t.Error("blue bytes must be zero")
	}
	if buf[7] != protocol.LEDChOrangeSelect {
		t.Errorf("orange ch select = 0x%02x, want 0xFF", buf[7])
	}
	if buf[8] != 0x0F {
		t.Errorf("orange level = 0x%02x, want 0x0F (masked)", buf[8])
	}
}

func TestLEDQueryVendorUnique(t *testing.T) {
	buf := LEDQuery(0x007D, 0xEE).Encode()
	if v := binary.LittleEndian.Uint16(buf[3:5]); v != protocol.VendorLEDQuery {
		t.Errorf("LED query vendor = 0x%04x, want 0x696B", v)
	}
	if buf[5] != 0xEE {
		t.Errorf("gw_mac5 routing tag = 0x%02x, want 0xEE", buf[5])
	}
}

func TestFindMeStartStop(t *testing.T) {
	start := FindMe(0x007D, true).Encode()
	stop := FindMe(0x007D, false).Encode()
	if start[5] == 0 {
		t.Error("find-me start mode byte should be non-zero")
	}
	if stop[5] != 0 || stop[6] != 0 {
		t.Errorf("find-me stop bytes should be zero, got %02x %02x", stop[5], stop[6])
	}
}

func TestWriteAlarmFragments(t *testing.T) {
	var rec [16]byte
	for i := range rec {
		rec[i] = byte(i + 1)
	}
	frags := WriteAlarm(rec)

	if frags[0].Data[0] != 0x00 {
		t.Errorf("frag 0 index byte = 0x%02x, want 0x00", frags[0].Data[0])
	}
	if frags[1].Data[0] != 0x01 {
		t.Errorf("frag 1 index byte = 0x%02x, want 0x01", frags[1].Data[0])
	}
	if !bytes.Equal(frags[0].Data[1:10], rec[0:9]) {
		t.Errorf("frag 0 payload mismatch")
	}
	if !bytes.Equal(frags[1].Data[1:8], rec[9:16]) {
		t.Errorf("frag 1 payload mismatch")
	}

	var wantXor byte
	for _, b := range rec {
		wantXor ^= b
	}
	if got := frags[1].Data[8]; got != wantXor {
		t.Errorf("xor checksum = 0x%02x, want 0x%02x", got, wantXor)
	}

	// Destination is the schedule coordinator.
	if frags[0].Destination != ScheduleCoordinator {
		t.Errorf("frag 0 dst = 0x%04x, want 0x0030", frags[0].Destination)
	}
}

func TestDeleteAlarmEncode(t *testing.T) {
	buf := DeleteAlarm(0x05, 0xEE).Encode()
	if buf[5] != 0x05 {
		t.Errorf("slot = 0x%02x, want 0x05", buf[5])
	}
	if buf[6] != 0xEE {
		t.Errorf("gw_mac5 = 0x%02x, want 0xEE", buf[6])
	}
}
