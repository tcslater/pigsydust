# piggsydust/ble

Reference BLE transport for [piggsydust](https://github.com/tcslater/piggsydust) using [`tinygo.org/x/bluetooth`](https://tinygo.org/x/bluetooth).

[![Go Reference](https://pkg.go.dev/badge/github.com/tcslater/piggsydust/ble.svg)](https://pkg.go.dev/github.com/tcslater/piggsydust/ble)

## Install

```bash
go get github.com/tcslater/piggsydust/ble
```

This is a **separate Go module** so the core `piggsydust` library stays free of BLE dependencies. Only import this package if you want to use the tinygo bluetooth stack.

## What's included

| Type | Purpose |
|------|---------|
| `Adapter` | Wraps `bluetooth.Adapter` — enable, scan, connect |
| `Scanner` (via `Adapter.Scan`) | Discovers Pixie devices with filtering by mesh name, network ID, gateway role |
| `Connection` | Holds a connected device with discovered GATT characteristics |
| `Transport` | Implements `piggsydust.Transport` — ready to pass to `piggsydust.NewClient` |

## Usage

```go
adapter, _ := ble.NewAdapter()

// Scan for gateway devices on the "Smart Light" mesh.
results, _ := adapter.Scan(ctx, piggsydust.ScanFilter{
    MeshName:    "Smart Light",
    GatewayOnly: true,
})
result := <-results
adapter.StopScan()

// Connect and discover GATT characteristics.
conn, _ := adapter.Connect(ctx, result.Advertisement, result.Address)
defer conn.Close()

// Hand the transport to piggsydust.
client := piggsydust.NewClient(ble.NewTransport(conn))
client.Login(ctx, "Smart Light", "12345678")
client.TurnOn(ctx, piggsydust.AddressBroadcast)
```

## Platform notes

| Platform | MAC address source | Notes |
|----------|--------------------|-------|
| Linux (BlueZ) | Direct from BLE scan | Works out of the box |
| macOS / iOS | DIS Model Number characteristic | OS randomises BLE addresses; the transport reads the real MAC from the Device Information Service |
| Windows | Direct from BLE scan | Requires WinRT bluetooth support |

## GATT service details

The Telink mesh service UUID is `00010203-0405-0607-0809-0a0b0c0d1910` with four characteristics:

| Suffix | Name | Used for |
|--------|------|----------|
| `1911` | CHAR_NOTIFY | Encrypted notification subscription |
| `1912` | CHAR_CMD | Encrypted command writes |
| `1913` | CHAR_OTA | OTA/config (unused in normal operation) |
| `1914` | CHAR_PAIR | Login handshake and heartbeat keepalive |
