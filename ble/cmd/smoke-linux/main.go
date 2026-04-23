//go:build linux

// smoke-linux exercises the BlueZ-backed pigsydust/ble path end-to-end on
// a Linux host: scan → connect → login → subscribe notify → send SetUTC →
// disconnect. No lights are toggled; the goal is to confirm the D-Bus path
// handles Telink's no-CCCD notifications and ATT Write Request/Command
// distinctions correctly.
//
// Target: a host running bluetoothd 5.49+ with a compatible HCI adapter.
//
// Usage:
//
//	./smoke-linux -password '<mesh password>'
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/tcslater/pigsydust"
	"github.com/tcslater/pigsydust/ble"
	"github.com/tcslater/pigsydust/command"
)

func main() {
	meshName := flag.String("mesh-name", "Smart Light", "BLE mesh name")
	meshPassword := flag.String("password", os.Getenv("PIGSY_MESH_PASSWORD"), "mesh password (or set PIGSY_MESH_PASSWORD)")
	scanTimeout := flag.Duration("scan-timeout", 15*time.Second, "BLE scan timeout")
	observe := flag.Duration("observe", 5*time.Second, "how long to observe notifications after kick")
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
	_ = adapter.StopScan()
	fmt.Printf("      picked %s RSSI=%d mac=%s type=0x%02X\n",
		best.Address.String(), best.RSSI, best.Advertisement.MAC,
		best.Advertisement.DeviceType)

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

	fmt.Println("[3/5] Subscribing to notifications...")
	notifyCtx, notifyCancel := context.WithCancel(ctx)
	defer notifyCancel()
	notifications, err := client.Notifications(notifyCtx)
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	rx := 0
	done := make(chan struct{})
	go func() {
		defer close(done)
		for n := range notifications {
			rx++
			fmt.Printf("      rx opcode=0x%02X src=0x%04X payload=%x\n",
				n.Opcode, n.Source, n.Payload)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	fmt.Println("[4/5] Sending SetUTC...")
	if err := client.Send(ctx, command.SetUTC(time.Now())); err != nil {
		log.Fatalf("set_utc: %v", err)
	}

	fmt.Printf("[5/5] Observing for %s...\n", *observe)
	select {
	case <-ctx.Done():
	case <-time.After(*observe):
	}

	notifyCancel()
	<-done
	fmt.Printf("done. %d notifications observed.\n", rx)
}
