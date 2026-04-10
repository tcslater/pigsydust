// Package piggsydust provides a Go client for controlling SAL Pixie / Telink
// BLE mesh wall switches.
//
// Pixie switches use a proprietary BLE mesh protocol with AES-CCM encryption.
// All communication is fully offline — no cloud, no hub, no app dependency.
// A client connects to any mesh node (the "gateway"), authenticates with
// shared mesh credentials, and sends encrypted commands that the mesh relays
// to all other nodes.
//
// # Architecture
//
// The library is split into protocol logic and BLE transport:
//
//   - Protocol: encryption, packet construction, session management (this package)
//   - Transport: BLE scanning, GATT I/O (user-provided via the [Transport] interface)
//
// Users implement the [Transport] interface using their preferred BLE library
// (e.g. tinygo.org/x/bluetooth, github.com/go-ble/ble, CoreBluetooth bindings)
// and pass it to [NewClient].
//
// # Quick Start
//
//	// 1. Connect to a Pixie device using your BLE library.
//	// 2. Build a Transport from the connected peripheral.
//	transport := myBLEAdapter.Connect(ctx, peripheral)
//
//	// 3. Create a client and login.
//	client := piggsydust.NewClient(transport,
//	    piggsydust.WithLogger(slog.Default()),
//	)
//	err := client.Login(ctx, "Smart Light", "12345678")
//
//	// 4. Control devices.
//	client.TurnOn(ctx, piggsydust.AddressBroadcast)
//	client.SetLEDOrange(ctx, piggsydust.Address(1), 15)
//
//	// 5. Clean up.
//	client.Close()
//
// # Subpackages
//
//   - [github.com/tcslater/piggsydust/crypto]: AES-CCM encryption primitives
//   - [github.com/tcslater/piggsydust/command]: protocol command builders
//   - [github.com/tcslater/piggsydust/schedule]: alarm record construction and timezone conversion
package piggsydust
