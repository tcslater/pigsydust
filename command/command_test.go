package command

import (
	"testing"
	"time"
)

func TestOnOff_Encode(t *testing.T) {
	cmd := OnOff(AddrBroadcast, true)
	buf := cmd.Encode()

	if len(buf) != 15 {
		t.Fatalf("length: got %d, want 15", len(buf))
	}

	// dst = 0xFFFF little-endian
	if buf[0] != 0xFF || buf[1] != 0xFF {
		t.Errorf("dst: got %02x %02x, want ff ff", buf[0], buf[1])
	}

	// opcode = (3 << 6) | (0xed & 0x3f) = 0xc0 | 0x2d = 0xed
	if buf[2] != 0xed {
		t.Errorf("opcode: got 0x%02x, want 0xed", buf[2])
	}

	// vendor = 0x6969 little-endian
	if buf[3] != 0x69 || buf[4] != 0x69 {
		t.Errorf("vendor: got %02x %02x, want 69 69", buf[3], buf[4])
	}

	// state = 0x01 (ON)
	if buf[5] != 0x01 {
		t.Errorf("state: got 0x%02x, want 0x01", buf[5])
	}

	// rest should be zero-padded
	for i := 6; i < 15; i++ {
		if buf[i] != 0 {
			t.Errorf("padding[%d]: got 0x%02x, want 0x00", i, buf[i])
		}
	}
}

func TestOnOff_Off(t *testing.T) {
	cmd := OnOff(0x0001, false)
	buf := cmd.Encode()

	if buf[5] != 0x00 {
		t.Errorf("state: got 0x%02x, want 0x00", buf[5])
	}
}

func TestStatusPoll_Encode(t *testing.T) {
	cmd := StatusPoll(AddrBroadcastPoll)
	buf := cmd.Encode()

	if len(buf) != 7 {
		t.Fatalf("length: got %d, want 7", len(buf))
	}

	// dst = 0x7FFF
	if buf[0] != 0xFF || buf[1] != 0x7F {
		t.Errorf("dst: got %02x %02x, want ff 7f", buf[0], buf[1])
	}

	// vendor = 0x0211
	if buf[3] != 0x11 || buf[4] != 0x02 {
		t.Errorf("vendor: got %02x %02x, want 11 02", buf[3], buf[4])
	}
}

func TestStatusQuery_Encode(t *testing.T) {
	cmd := StatusQuery()
	buf := cmd.Encode()

	if len(buf) != 10 {
		t.Fatalf("length: got %d, want 10", len(buf))
	}
}

func TestSetUTC_Encode(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cmd := SetUTC(now)
	buf := cmd.Encode()

	if len(buf) != 15 {
		t.Fatalf("length: got %d, want 15", len(buf))
	}

	// Timezone byte must be 0x00.
	if buf[9] != 0x00 {
		t.Errorf("timezone byte: got 0x%02x, want 0x00", buf[9])
	}
}

func TestLEDSetBlue_Encode(t *testing.T) {
	cmd := LEDSetBlue(0x0001, true)
	buf := cmd.Encode()

	if len(buf) != 15 {
		t.Fatalf("length: got %d, want 15", len(buf))
	}

	// b_ch=0xa0, b_lvl=0x12, o_ch=0x00, o_lvl=0x00
	if buf[5] != 0xa0 {
		t.Errorf("b_ch: got 0x%02x, want 0xa0", buf[5])
	}
	if buf[6] != 0x12 {
		t.Errorf("b_lvl: got 0x%02x, want 0x12", buf[6])
	}
	if buf[7] != 0x00 || buf[8] != 0x00 {
		t.Errorf("orange should be zeroed: got %02x %02x", buf[7], buf[8])
	}
}

func TestLEDSetOrange_Encode(t *testing.T) {
	cmd := LEDSetOrange(0x0001, 15)
	buf := cmd.Encode()

	// o_ch=0xff, o_lvl=0x0f
	if buf[7] != 0xff {
		t.Errorf("o_ch: got 0x%02x, want 0xff", buf[7])
	}
	if buf[8] != 0x0f {
		t.Errorf("o_lvl: got 0x%02x, want 0x0f", buf[8])
	}
	// Blue should be zeroed.
	if buf[5] != 0x00 || buf[6] != 0x00 {
		t.Errorf("blue should be zeroed: got %02x %02x", buf[5], buf[6])
	}
}

func TestLEDSetOrange_ClampsBrightness(t *testing.T) {
	cmd := LEDSetOrange(0x0001, 0xFF)
	buf := cmd.Encode()

	// Only lower nibble should be used.
	if buf[8] != 0x0F {
		t.Errorf("o_lvl: got 0x%02x, want 0x0f (clamped)", buf[8])
	}
}

func TestFindMe_Start(t *testing.T) {
	cmd := FindMe(0x0001, true)
	buf := cmd.Encode()

	if buf[5] != 0x03 || buf[6] != 0x0f {
		t.Errorf("start: got mode=0x%02x duration=0x%02x, want 0x03 0x0f", buf[5], buf[6])
	}
}

func TestFindMe_Stop(t *testing.T) {
	cmd := FindMe(0x0001, false)
	buf := cmd.Encode()

	if buf[5] != 0x00 || buf[6] != 0x00 {
		t.Errorf("stop: got mode=0x%02x duration=0x%02x, want 0x00 0x00", buf[5], buf[6])
	}
}

func TestGroupOnOff_Encode(t *testing.T) {
	group := uint16(0x8001) // group 1
	cmd := GroupOnOff(group, true)
	buf := cmd.Encode()

	if len(buf) != 15 {
		t.Fatalf("length: got %d, want 15", len(buf))
	}

	// state = 0x0e (ON)
	if buf[5] != 0x0e {
		t.Errorf("state: got 0x%02x, want 0x0e", buf[5])
	}

	// Group address appears in data tail at offset 11-12.
	if buf[11] != 0x01 || buf[12] != 0x80 {
		t.Errorf("group tail: got %02x %02x, want 01 80", buf[11], buf[12])
	}
}
