// Smoke test for the rewritten pigsydust Go library.
//
// Scans for the Pixie mesh, connects, logs in, subscribes to notifications,
// broadcasts a status query, then toggles the mudroom light (device 39)
// on and off once, restoring its original state.
//
// Exercises every layer: crypto (login + command encrypt + notification
// decrypt), transport (ble/), command builders, notification parsers.
//
// Usage:
//
//	go run ./cmd/smoke -password '<mesh password>'
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/tcslater/pigsydust"
	"github.com/tcslater/pigsydust/ble"
	"github.com/tcslater/pigsydust/command"
	"github.com/tcslater/pigsydust/protocol"
)

const (
	mudroomAddr uint16 = 9
	mudroomName        = "Mud Room"
)

func main() {
	meshName := flag.String("mesh-name", "Smart Light", "BLE mesh name")
	meshPassword := flag.String("password", os.Getenv("PIGSY_MESH_PASSWORD"), "mesh password (or set PIGSY_MESH_PASSWORD)")
	scanTimeout := flag.Duration("scan-timeout", 10*time.Second, "BLE scan timeout")
	flag.Parse()

	if *meshPassword == "" {
		log.Fatal("mesh password required via -password or PIGSY_MESH_PASSWORD")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	adapter, err := ble.NewAdapter()
	if err != nil {
		log.Fatalf("adapter init: %v", err)
	}

	// [1/5] Scan — pick strongest gateway advertising the right mesh.
	fmt.Printf("[1/5] Scanning for mesh %q (up to %s)...\n", *meshName, *scanTimeout)
	scanCtx, scanCancel := context.WithTimeout(ctx, *scanTimeout)
	defer scanCancel()

	results, err := adapter.Scan(scanCtx, pigsydust.ScanFilter{MeshName: *meshName})
	if err != nil {
		log.Fatalf("scan start: %v", err)
	}

	var best *ble.ScanResult
	for r := range results {
		r := r
		if best == nil || r.RSSI > best.RSSI {
			best = &r
		}
		if r.Advertisement.DeviceType == pigsydust.DeviceTypeGateway {
			best = &r
			break
		}
	}
	if best == nil {
		log.Fatal("no matching mesh node found")
	}
	fmt.Printf("      picked %s RSSI=%d mac=%s type=0x%02X\n",
		best.Address.String(), best.RSSI, best.Advertisement.MAC,
		best.Advertisement.DeviceType)

	// [2/5] Connect + login.
	fmt.Println("[2/5] Connecting...")
	conn, err := adapter.Connect(ctx, best.Advertisement, best.Address)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	transport := ble.NewTransport(conn)
	client := pigsydust.NewClient(transport)
	defer client.Close(context.Background())

	fmt.Printf("      logging in (gw=%s)...\n", transport.GatewayMAC())
	if err := client.Login(ctx, *meshName, *meshPassword); err != nil {
		log.Fatalf("login: %v", err)
	}
	fmt.Println("      logged in.")

	// [3/5] Subscribe and collect statuses.
	fmt.Println("[3/5] Subscribing to notifications...")
	notifyCtx, notifyCancel := context.WithCancel(ctx)
	defer notifyCancel()
	notifications, err := client.Notifications(notifyCtx)
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	statuses := make(map[uint16]pigsydust.DeviceStatus)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for n := range notifications {
			switch n.Opcode {
			case protocol.OpNotifyStatusPoll:
				if ds, err := pigsydust.ParseDeviceStatus(n); err == nil {
					statuses[ds.Address] = ds
				}
			case protocol.OpNotifyStatusBroadcast:
				if list, err := pigsydust.ParseDeviceStatusBroadcast(n); err == nil {
					for _, ds := range list {
						statuses[ds.Address] = ds
					}
				}
			default:
				fmt.Printf("      rx opcode=0x%02X src=0x%04X payload=%x\n",
					n.Opcode, n.Source, n.Payload)
			}
		}
	}()

	// Small settling delay so CoreBluetooth actually lands the CCCD write
	// from EnableNotifications before we start firing commands.
	time.Sleep(500 * time.Millisecond)

	// Python client always sends set_utc right after login. Mimic that.
	fmt.Println("      sending set_utc...")
	if err := client.Send(ctx, command.SetUTC(time.Now())); err != nil {
		log.Fatalf("set_utc: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	fmt.Println("      querying status (3s collect window)...")
	if err := client.Send(ctx, command.StatusQuery()); err != nil {
		log.Fatalf("status query: %v", err)
	}
	time.Sleep(3 * time.Second)

	fmt.Println("      blind toggle mudroom (ON, no status needed)...")
	if err := client.Send(ctx, command.OnOff(mudroomAddr, true)); err != nil {
		log.Fatalf("blind on: %v", err)
	}
	time.Sleep(2 * time.Second)
	fmt.Println("      blind toggle mudroom (OFF)...")
	if err := client.Send(ctx, command.OnOff(mudroomAddr, false)); err != nil {
		log.Fatalf("blind off: %v", err)
	}
	time.Sleep(2 * time.Second)

	addrs := make([]uint16, 0, len(statuses))
	for a := range statuses {
		addrs = append(addrs, a)
	}
	sort.Slice(addrs, func(i, j int) bool { return addrs[i] < addrs[j] })

	fmt.Printf("      %d devices responded:\n", len(addrs))
	for _, a := range addrs {
		ds := statuses[a]
		state := "OFF"
		if ds.On {
			state = "ON"
		}
		tag := ""
		if a == mudroomAddr {
			tag = "  ← " + mudroomName
		}
		fmt.Printf("        addr=%3d  %s  type=0x%02X/0x%02X%s\n",
			a, state, ds.DeviceType, ds.DeviceSubtype, tag)
	}

	mudroom, found := statuses[mudroomAddr]
	if !found {
		fmt.Printf("      WARNING: mudroom (addr=%d) did not respond — skipping toggle\n", mudroomAddr)
		notifyCancel()
		<-done
		return
	}
	originallyOn := mudroom.On

	// [4/5] Toggle.
	fmt.Printf("[4/5] Mudroom currently %v — toggling opposite...\n", onOff(originallyOn))
	if err := client.Send(ctx, command.OnOff(mudroomAddr, !originallyOn)); err != nil {
		log.Fatalf("toggle: %v", err)
	}
	time.Sleep(2 * time.Second)

	if ds, ok := statuses[mudroomAddr]; ok {
		fmt.Printf("      observed state: %v\n", onOff(ds.On))
		if ds.On == originallyOn {
			fmt.Println("      WARNING: state did not change in unsolicited notification")
		}
	}

	// [5/5] Restore.
	fmt.Printf("[5/5] Restoring mudroom to %v...\n", onOff(originallyOn))
	if err := client.Send(ctx, command.OnOff(mudroomAddr, originallyOn)); err != nil {
		log.Fatalf("restore: %v", err)
	}
	time.Sleep(2 * time.Second)

	notifyCancel()
	<-done
	fmt.Println("done.")
}

func onOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}
