package pigsydust

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/tcslater/pigsydust/command"
	"github.com/tcslater/pigsydust/crypto"
	"github.com/tcslater/pigsydust/schedule"
)

// Client manages an authenticated session with a Pixie BLE mesh.
//
// Create a Client with [NewClient], then call [Client.Login] to authenticate.
// All control methods require a successful login first.
type Client struct {
	transport Transport
	cfg       clientConfig
	sess      session

	notifications chan Notification
	cancel        context.CancelFunc
	done          chan struct{}
}

// NewClient creates a new Client using the given transport.
// Call [Client.Login] to authenticate before sending commands.
func NewClient(t Transport, opts ...Option) *Client {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return &Client{
		transport:     t,
		cfg:           cfg,
		notifications: make(chan Notification, 64),
	}
}

// Login authenticates with the mesh and starts the notification listener
// and heartbeat goroutines.
//
// The full sequence is:
//  1. Generate random nonce (randA)
//  2. Write login request to CHAR_PAIR
//  3. Read login response from CHAR_PAIR
//  4. Derive session key
//  5. Subscribe to notifications
//  6. Send time synchronisation broadcast
func (c *Client) Login(ctx context.Context, meshName, meshPassword string) error {
	// Generate randA.
	var randA [8]byte
	r := c.cfg.randSource
	if r == nil {
		r = rand.Reader
	}
	if _, err := io.ReadFull(r, randA[:]); err != nil {
		return fmt.Errorf("pigsydust: generating login nonce: %w", err)
	}

	// Generate session salt.
	var salt [2]byte
	if _, err := io.ReadFull(r, salt[:]); err != nil {
		return fmt.Errorf("pigsydust: generating session salt: %w", err)
	}

	// Build and send login request.
	req := crypto.LoginRequest(meshName, meshPassword, randA)
	if err := c.transport.WritePair(ctx, req[:]); err != nil {
		return fmt.Errorf("pigsydust: writing login request: %w", err)
	}

	// Read login response.
	resp, err := c.transport.ReadPair(ctx)
	if err != nil {
		return fmt.Errorf("pigsydust: reading login response: %w", err)
	}

	randB, err := crypto.ParseLoginResponse(resp)
	if err != nil {
		return err
	}

	// Derive session key.
	sk := crypto.DeriveSessionKey(meshName, meshPassword, randA, randB)

	c.sess = session{
		sessionKey:  sk,
		gwMAC:       c.transport.GatewayMAC(),
		sessionSalt: salt,
	}

	// Subscribe to notifications and start listener.
	notifyCh, err := c.transport.SubscribeNotify(ctx)
	if err != nil {
		return fmt.Errorf("pigsydust: subscribing to notifications: %w", err)
	}

	listenerCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan struct{})

	go c.notificationListener(listenerCtx, notifyCh)
	go c.heartbeatLoop(listenerCtx)

	// Send time sync.
	if err := c.send(ctx, command.SetUTC(time.Now())); err != nil {
		c.cfg.logger.Warn("failed to send time sync", "err", err)
	}

	return nil
}

// Close shuts down the heartbeat and notification listener goroutines.
func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
		<-c.done
	}
	return nil
}

// Notifications returns a read-only channel of decoded notifications
// that were not consumed by request-response methods.
func (c *Client) Notifications() <-chan Notification {
	return c.notifications
}

// TurnOn sends an on command to the given address.
func (c *Client) TurnOn(ctx context.Context, dst Address) error {
	return c.send(ctx, command.OnOff(uint16(dst), true))
}

// TurnOff sends an off command to the given address.
func (c *Client) TurnOff(ctx context.Context, dst Address) error {
	return c.send(ctx, command.OnOff(uint16(dst), false))
}

// GroupTurnOn sends a group on command.
func (c *Client) GroupTurnOn(ctx context.Context, group Address) error {
	return c.send(ctx, command.GroupOnOff(uint16(group), true))
}

// GroupTurnOff sends a group off command.
func (c *Client) GroupTurnOff(ctx context.Context, group Address) error {
	return c.send(ctx, command.GroupOnOff(uint16(group), false))
}

// QueryStatus broadcasts a status query and returns a channel that
// delivers [BroadcastDeviceStatus] values as they arrive. Each 0xdc
// notification packs up to two devices, which are delivered individually
// on the channel. The channel closes after the client's command timeout.
func (c *Client) QueryStatus(ctx context.Context) (<-chan BroadcastDeviceStatus, error) {
	if err := c.send(ctx, command.StatusQuery()); err != nil {
		return nil, err
	}

	out := make(chan BroadcastDeviceStatus, 64)
	go func() {
		defer close(out)

		timeout := time.After(c.cfg.commandTimeout)
		for {
			ch := c.sess.registerWaiter(0xdc, 0)
			select {
			case n := <-ch:
				c.sess.unregisterWaiter(0xdc, 0)
				devices, err := ParseBroadcastStatus(n)
				if err != nil {
					c.cfg.logger.Warn("bad broadcast status notification", "err", err)
					continue
				}
				for _, ds := range devices {
					select {
					case out <- ds:
					case <-timeout:
						return
					case <-ctx.Done():
						return
					}
				}
			case <-timeout:
				c.sess.unregisterWaiter(0xdc, 0)
				return
			case <-ctx.Done():
				c.sess.unregisterWaiter(0xdc, 0)
				return
			}
		}
	}()

	return out, nil
}

// PollStatus sends a unicast status poll and waits for the response.
func (c *Client) PollStatus(ctx context.Context, dst Address) (DeviceStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.commandTimeout)
	defer cancel()

	if err := c.send(ctx, command.StatusPoll(uint16(dst))); err != nil {
		return DeviceStatus{}, err
	}

	n, err := c.sess.waitForNotification(ctx, 0xdb, dst)
	if err != nil {
		return DeviceStatus{}, &OpError{Op: "PollStatus", Addr: dst, Err: err}
	}
	return ParseDeviceStatus(n)
}

// SetGroups sets a device's complete group membership list.
func (c *Client) SetGroups(ctx context.Context, dst Address, groups []uint8) error {
	gwMAC5 := c.sess.gwMAC.GatewayMAC5()
	return c.send(ctx, command.SetGroupMembership(uint16(dst), groups, gwMAC5))
}

// QueryGroups queries a device's group membership.
func (c *Client) QueryGroups(ctx context.Context, dst Address) (GroupMembership, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.commandTimeout)
	defer cancel()

	if err := c.send(ctx, command.QueryGroupMembership(uint16(dst))); err != nil {
		return GroupMembership{}, err
	}

	n, err := c.sess.waitForNotification(ctx, 0xd4, dst)
	if err != nil {
		return GroupMembership{}, &OpError{Op: "QueryGroups", Addr: dst, Err: err}
	}
	return ParseGroupMembership(n)
}

// SetLEDBlue controls the blue LED indicator channel.
func (c *Client) SetLEDBlue(ctx context.Context, dst Address, on bool) error {
	return c.send(ctx, command.LEDSetBlue(uint16(dst), on))
}

// SetLEDOrange controls the orange LED indicator channel.
// Level is brightness 0-15 (0 = off).
func (c *Client) SetLEDOrange(ctx context.Context, dst Address, level uint8) error {
	return c.send(ctx, command.LEDSetOrange(uint16(dst), level))
}

// QueryLED queries the LED indicator state of a device.
//
// This automatically performs the required wake-up sequence:
//  1. Send status poll
//  2. Wait for status response
//  3. Wait 210ms settling time
//  4. Send LED query
//  5. Return LED state response
func (c *Client) QueryLED(ctx context.Context, dst Address) (LEDState, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.commandTimeout)
	defer cancel()

	// Wake-up: poll status first.
	if _, err := c.PollStatus(ctx, dst); err != nil {
		return LEDState{}, &OpError{Op: "QueryLED", Addr: dst, Err: fmt.Errorf("wake-up poll: %w", err)}
	}

	// Settling time.
	select {
	case <-time.After(210 * time.Millisecond):
	case <-ctx.Done():
		return LEDState{}, &OpError{Op: "QueryLED", Addr: dst, Err: ctx.Err()}
	}

	gwMAC5 := c.sess.gwMAC.GatewayMAC5()
	if err := c.send(ctx, command.LEDQuery(uint16(dst), gwMAC5)); err != nil {
		return LEDState{}, err
	}

	n, err := c.sess.waitForNotification(ctx, 0xd3, dst)
	if err != nil {
		return LEDState{}, &OpError{Op: "QueryLED", Addr: dst, Err: err}
	}
	return ParseLEDState(n)
}

// FindMe starts or stops the find-me LED blink on a device.
func (c *Client) FindMe(ctx context.Context, dst Address, start bool) error {
	return c.send(ctx, command.FindMe(uint16(dst), start))
}

// SyncTime broadcasts the current time to all mesh devices. This is
// called automatically during [Client.Login], but can be called again
// to re-sync the mesh clock during long-running sessions.
func (c *Client) SyncTime(ctx context.Context) error {
	return c.send(ctx, command.SetUTC(time.Now()))
}

// CreateAlarm writes an alarm record to the mesh coordinator and returns
// the assigned slot index.
//
// Alarm times are interpreted relative to the mesh clock, which is set
// once during [Client.Login] via a time sync broadcast. If the session
// has been running for a long time, call [Client.SyncTime] before
// creating alarms to ensure the mesh clock is accurate.
func (c *Client) CreateAlarm(ctx context.Context, record schedule.AlarmRecord) (uint8, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.commandTimeout)
	defer cancel()

	data, err := record.MarshalBinary()
	if err != nil {
		return 0, err
	}

	var alarmBytes [16]byte
	copy(alarmBytes[:], data)

	frags := command.WriteAlarm(alarmBytes)
	if err := c.send(ctx, frags[0]); err != nil {
		return 0, err
	}
	if err := c.send(ctx, frags[1]); err != nil {
		return 0, err
	}

	// Query the assigned slot.
	gwMAC5 := c.sess.gwMAC.GatewayMAC5()
	if err := c.send(ctx, command.SlotQuery(gwMAC5)); err != nil {
		return 0, err
	}

	n, err := c.sess.waitForNotification(ctx, 0xd3, AddressScheduleCoordinator)
	if err != nil {
		return 0, &OpError{Op: "CreateAlarm", Addr: AddressScheduleCoordinator, Err: err}
	}

	return ParseSlotAssignment(n)
}

// ListAlarms walks the alarm slots on the coordinator and returns all
// stored alarms. Use target=0 to list all alarms, or a specific address
// to filter.
func (c *Client) ListAlarms(ctx context.Context, target Address) ([]schedule.StoredAlarm, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.commandTimeout*5) // walks can take time
	defer cancel()

	gwMAC5 := c.sess.gwMAC.GatewayMAC5()
	var alarms []schedule.StoredAlarm
	cursor := uint8(0)

	for {
		if err := c.send(ctx, command.QueryAlarm(cursor, gwMAC5, uint16(target))); err != nil {
			return alarms, err
		}

		// Receive two 0xc2 fragments.
		n0, err := c.sess.waitForNotification(ctx, 0xc2, AddressScheduleCoordinator)
		if err != nil {
			return alarms, &OpError{Op: "ListAlarms", Addr: AddressScheduleCoordinator, Err: err}
		}

		slot0, frag0, err := ParseAlarmFragment(n0)
		if err != nil {
			return alarms, err
		}
		if slot0 == 0xff {
			break // end of list
		}

		n1, err := c.sess.waitForNotification(ctx, 0xc2, AddressScheduleCoordinator)
		if err != nil {
			return alarms, &OpError{Op: "ListAlarms", Addr: AddressScheduleCoordinator, Err: err}
		}

		_, frag1, err := ParseAlarmFragment(n1)
		if err != nil {
			return alarms, err
		}

		// Reassemble the 16-byte alarm record from two fragments.
		var recordBytes [16]byte
		copy(recordBytes[:], frag0)
		copy(recordBytes[len(frag0):], frag1)

		var record schedule.AlarmRecord
		if err := record.UnmarshalBinary(recordBytes[:]); err != nil {
			return alarms, err
		}

		alarms = append(alarms, schedule.StoredAlarm{Slot: slot0, Alarm: record})
		cursor = slot0 + 1
	}

	return alarms, nil
}

// DeleteAlarm deletes an alarm at the given slot index.
func (c *Client) DeleteAlarm(ctx context.Context, slot uint8) error {
	gwMAC5 := c.sess.gwMAC.GatewayMAC5()
	return c.send(ctx, command.DeleteAlarm(slot, gwMAC5))
}

// send encrypts and writes a command to the mesh.
func (c *Client) send(ctx context.Context, cmd command.Command) error {
	plaintext := cmd.Encode()
	sno := c.sess.nextSNO()
	nonce := crypto.CommandNonce(c.sess.gwMAC, sno)
	packet := crypto.Encrypt(c.sess.sessionKey, nonce, sno, plaintext)

	if err := c.transport.WriteCommand(ctx, packet); err != nil {
		return &OpError{Op: "send", Addr: Address(cmd.Destination), Err: err}
	}

	return nil
}

// notificationListener decrypts and routes incoming notifications.
func (c *Client) notificationListener(ctx context.Context, rawCh <-chan []byte) {
	defer close(c.done)

	for {
		select {
		case raw, ok := <-rawCh:
			if !ok {
				return
			}

			sno, srcAddr, tag, ciphertext, err := ParseNotificationWire(raw)
			if err != nil {
				c.cfg.logger.Warn("bad notification wire format", "err", err)
				continue
			}

			nonce := crypto.NotificationNonce(c.sess.gwMAC, sno, srcAddr)
			plaintext, err := crypto.Decrypt(c.sess.sessionKey, nonce, tag, ciphertext)
			if err != nil {
				c.cfg.logger.Warn("notification decryption failed", "err", err, "src", srcAddr)
				continue
			}

			n, err := ParseNotification(Address(srcAddr), plaintext)
			if err != nil {
				c.cfg.logger.Warn("notification parse failed", "err", err)
				continue
			}

			c.cfg.logger.Debug("notification received",
				"src", n.Source, "opcode", fmt.Sprintf("0x%02x", n.Opcode))

			// Try to route to a waiting request-response first.
			if !c.sess.routeNotification(n) {
				// Unmatched — send to the public notifications channel.
				select {
				case c.notifications <- n:
				default:
					c.cfg.logger.Warn("notification channel full, dropping", "opcode", n.Opcode)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// heartbeatLoop periodically reads from CHAR_PAIR to keep the BLE
// connection alive.
func (c *Client) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, err := c.transport.ReadPair(readCtx)
			cancel()
			if err != nil {
				c.cfg.logger.Warn("heartbeat read failed", "err", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
