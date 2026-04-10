package piggsydust_test

import (
	"fmt"

	"github.com/tcslater/piggsydust"
	"github.com/tcslater/piggsydust/schedule"
)

func ExampleGroupAddress() {
	addr := piggsydust.GroupAddress(3)
	fmt.Println(addr)
	fmt.Println(addr.IsGroup())

	id, ok := addr.GroupID()
	fmt.Println(id, ok)
	// Output:
	// group-3
	// true
	// 3 true
}

func ExampleParseMAC() {
	mac, err := piggsydust.ParseMAC("AA:BB:CC:DD:EE:FF")
	if err != nil {
		panic(err)
	}
	fmt.Println(mac)
	fmt.Println("GatewayMAC5:", fmt.Sprintf("0x%02X", mac.GatewayMAC5()))
	// Output:
	// AA:BB:CC:DD:EE:FF
	// GatewayMAC5: 0xFF
}

func ExampleNewRecurringAlarm() {
	// Create a recurring alarm for 8:00 AM Mon-Fri in AEST (UTC+10).
	alarm := schedule.NewRecurringAlarm(
		0x01,                                          // alarm ID
		8, 0,                                          // 8:00 AM local
		schedule.Weekdays,                             // Mon-Fri
		10,                                            // AEST = UTC+10
		uint16(piggsydust.AddressBroadcast),           // target all devices
		schedule.ActionOnFull,                         // turn on at 100%
	)

	fmt.Printf("UTC hour: %d\n", alarm.Hour)
	fmt.Printf("UTC weekday mask: 0x%02x\n", alarm.Repeat)
	// Output:
	// UTC hour: 22
	// UTC weekday mask: 0x4f
}
