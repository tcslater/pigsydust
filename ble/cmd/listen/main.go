// Notification-visibility probe for sandy design.
//
// Logs in as a normal pigsydust client, subscribes to notifications, and
// dumps every packet for a fixed duration. No commands are sent apart from
// the mandatory set_utc. Purpose: confirm that unsolicited notifications
// (e.g. a wall switch toggled by hand) are delivered to a passive client,
// which is the load-bearing assumption for the sandy cross-building bridge.
//
// Usage:
//
//	go run ./cmd/listen -password '<mesh password>' -duration 30s
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
	"github.com/tcslater/pigsydust/protocol"
)

func main() {
	meshName := flag.String("mesh-name", "Smart Light", "BLE mesh name")
	meshPassword := flag.String("password", os.Getenv("PIGSY_MESH_PASSWORD"), "mesh password (or set PIGSY_MESH_PASSWORD)")
	scanTimeout := flag.Duration("scan-timeout", 10*time.Second, "BLE scan timeout")
	duration := flag.Duration("duration", 30*time.Second, "how long to listen after login")
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

	fmt.Printf("scanning for %q (up to %s)...\n", *meshName, *scanTimeout)
	scanCtx, scanCancel := context.WithTimeout(ctx, *scanTimeout)
	defer scanCancel()
	results, err := adapter.Scan(scanCtx, pigsydust.ScanFilter{MeshName: *meshName})
	if err != nil {
		log.Fatalf("scan: %v", err)
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
	fmt.Printf("picked %s RSSI=%d mac=%s\n", best.Address, best.RSSI, best.Advertisement.MAC)

	conn, err := adapter.Connect(ctx, best.Advertisement, best.Address)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	transport := ble.NewTransport(conn)
	client := pigsydust.NewClient(transport)
	defer client.Close(context.Background())

	if err := client.Login(ctx, *meshName, *meshPassword); err != nil {
		log.Fatalf("login: %v", err)
	}
	fmt.Println("logged in.")

	notifyCtx, notifyCancel := context.WithCancel(ctx)
	defer notifyCancel()
	notifications, err := client.Notifications(notifyCtx)
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	if err := client.Send(ctx, command.SetUTC(time.Now())); err != nil {
		log.Fatalf("set_utc: %v", err)
	}

	deadline := time.Now().Add(*duration)
	fmt.Printf("listening for %s — flip a switch by hand\n", *duration)

	counts := make(map[byte]int)
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			goto done
		case n, ok := <-notifications:
			if !ok {
				goto done
			}
			counts[n.Opcode]++
			t := time.Since(start).Truncate(time.Millisecond)
			name := opcodeName(n.Opcode)
			fmt.Printf("[%7s] opcode=0x%02X (%s) src=0x%04X payload=%x\n", t, n.Opcode, name, n.Source, n.Payload)
		case <-time.After(time.Until(deadline)):
			goto done
		}
		if time.Now().After(deadline) {
			break
		}
	}
done:
	fmt.Println("\n--- summary ---")
	for op, c := range counts {
		fmt.Printf("opcode=0x%02X (%s)  count=%d\n", op, opcodeName(op), c)
	}
}

func opcodeName(op byte) string {
	switch op {
	case protocol.OpNotifyStatusPoll:
		return "status-poll"
	case protocol.OpNotifyStatusBroadcast:
		return "status-broadcast"
	default:
		return "?"
	}
}
