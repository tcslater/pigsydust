// Package ble provides a reference [piggsydust.Transport] and [piggsydust.Scanner]
// implementation using [tinygo.org/x/bluetooth].
//
// This package is a separate Go module so that the core piggsydust library
// remains free of BLE dependencies. Import this package only if you want
// to use the tinygo bluetooth stack.
//
// # Quick Start
//
//	// Create and enable the BLE adapter.
//	adapter, err := ble.NewAdapter()
//
//	// Scan for Pixie mesh devices.
//	results, err := adapter.Scan(ctx, piggsydust.ScanFilter{
//	    MeshName:    "Smart Light",
//	    GatewayOnly: true,
//	})
//	result := <-results
//	adapter.StopScan()
//
//	// Connect and discover GATT services.
//	conn, err := adapter.Connect(ctx, result.Advertisement, result.Address)
//	defer conn.Close()
//
//	// Create a piggsydust client.
//	client := piggsydust.NewClient(ble.NewTransport(conn))
//	err = client.Login(ctx, "Smart Light", "12345678")
//	client.TurnOn(ctx, piggsydust.AddressBroadcast)
//
// # Platform Notes
//
// On macOS/iOS, the OS randomises BLE addresses. The transport automatically
// reads the real MAC from the Device Information Service Model Number
// characteristic (0x2a24) during [Adapter.Connect].
//
// On Linux, real MAC addresses are available directly from the BLE scan.
package ble
