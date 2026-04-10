<p align="center">
  <img src="splash.png" alt="PiggsyDust" width="400">
</p>

A Go library for controlling [SAL Pixie](https://pixieplus.com.au/) / Telink BLE mesh wall switches - fully offline, no cloud, no hub, no app dependency.

[![Go Reference](https://pkg.go.dev/badge/github.com/tcslater/piggsydust.svg)](https://pkg.go.dev/github.com/tcslater/piggsydust)

## Features

- **Full protocol implementation** - login, session key derivation, AES-CCM encryption/decryption
- **Device control** - on/off, group on/off, LED indicator control, find-me blink
- **Status monitoring** - broadcast queries, unicast polls, real-time notifications
- **Group management** - set membership, query groups, probe addresses
- **Schedules** - create/list/delete alarms, countdown timers, recurring schedules with timezone support
- **BLE-library agnostic** - implement the `Transport` interface with any BLE stack
- **Reference BLE transport** - ready-to-use adapter via [`tinygo.org/x/bluetooth`](https://tinygo.org/x/bluetooth) in the [`ble`](./ble) submodule

## Architecture

The library is split into two Go modules:

| Module | Import path | Purpose |
|--------|-------------|---------|
| Core | `github.com/tcslater/piggsydust` | Protocol logic, encryption, command building - zero BLE dependencies |
| BLE transport | `github.com/tcslater/piggsydust/ble` | Reference `Transport` implementation using tinygo bluetooth |

Users who prefer a different BLE library (go-ble, CoreBluetooth bindings, etc.) only need the core module and implement the 5-method `Transport` interface.

### Subpackages

| Package | Purpose |
|---------|---------|
| `piggsydust` | Client API, Transport interface, types, notification parsing |
| `piggsydust/crypto` | Reversed AES, login handshake, session keys, AES-CCM encrypt/decrypt |
| `piggsydust/command` | Protocol command builders for all opcodes |
| `piggsydust/schedule` | Alarm record construction, weekday bitmask, timezone conversion |

## Install

```bash
# Core library only (no BLE dependency)
go get github.com/tcslater/piggsydust

# With the tinygo BLE transport
go get github.com/tcslater/piggsydust/ble
```

## Quick start

```go
package main

import (
    "context"
    "log"

    "github.com/tcslater/piggsydust"
    "github.com/tcslater/piggsydust/ble"
)

func main() {
    ctx := context.Background()

    // 1. Enable the BLE adapter.
    adapter, err := ble.NewAdapter()
    if err != nil {
        log.Fatal(err)
    }

    // 2. Scan for Pixie devices.
    results, err := adapter.Scan(ctx, piggsydust.ScanFilter{
        MeshName:    "Smart Light",
        GatewayOnly: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    result := <-results
    adapter.StopScan()

    // 3. Connect and discover GATT services.
    conn, err := adapter.Connect(ctx, result.Advertisement, result.Address)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // 4. Create a client and authenticate.
    client := piggsydust.NewClient(ble.NewTransport(conn))
    if err := client.Login(ctx, "Smart Light", "12345678"); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 5. Control devices.
    client.TurnOn(ctx, piggsydust.AddressBroadcast)
}
```

## Implementing your own Transport

If you're using a different BLE library, implement the `Transport` interface:

```go
type Transport interface {
    // WritePair writes to CHAR_PAIR (0x1914) with ATT Write Request.
    WritePair(ctx context.Context, data []byte) error

    // ReadPair reads from CHAR_PAIR (0x1914).
    ReadPair(ctx context.Context) ([]byte, error)

    // WriteCommand writes an encrypted packet to CHAR_CMD (0x1912).
    WriteCommand(ctx context.Context, data []byte) error

    // SubscribeNotify subscribes to CHAR_NOTIFY (0x1911) and returns
    // a channel delivering raw 20-byte notification packets.
    SubscribeNotify(ctx context.Context) (<-chan []byte, error)

    // GatewayMAC returns the 6-byte MAC of the connected gateway.
    GatewayMAC() MACAddress
}
```

The Telink mesh GATT service UUID is `00010203-0405-0607-0809-0a0b0c0d1910` with characteristic suffixes `1911`-`1914`.

## Mesh credentials

All nodes in a Pixie mesh share two values:

- **Mesh name** - typically `"Smart Light"` (the firmware default)
- **Mesh password** - the numeric string shown in the Pixie app's "Share Home" screen

## Protocol reference

See [`docs/PROTOCOL-REFERENCE.md`](docs/PROTOCOL-REFERENCE.md) for the complete reverse-engineered protocol specification.

## License

MIT
