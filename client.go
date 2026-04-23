package pigsydust

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/tcslater/pigsydust/command"
	"github.com/tcslater/pigsydust/crypto"
)

// Client is an authenticated session with a single Pixie mesh node.
//
// The connected node serves as the entry point for the entire mesh —
// broadcasts relay to every other node, and individual-address commands are
// routed by the mesh firmware. One Client = one BLE connection.
//
// Lifecycle:
//
//  1. NewClient(transport, opts...) — wraps a connected transport.
//  2. Login(ctx, meshName, meshPassword) — pairs and derives the session key.
//  3. Send(ctx, cmd) for each command; Notifications(ctx) for a decrypted
//     notification stream.
//  4. Close(ctx) — stops the heartbeat loop. The caller is responsible for
//     closing the underlying Transport.
type Client struct {
	t    Transport
	opts clientOptions

	session session

	hbOnce   sync.Once
	hbCancel context.CancelFunc
	hbDone   chan struct{}

	notifyMu     sync.Mutex
	notifyActive bool
}

// NewClient wraps an already-connected [Transport].
func NewClient(t Transport, opts ...ClientOption) *Client {
	o := defaultOptions
	if o.logger == nil {
		o.logger = slog.Default()
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{
		t:       t,
		opts:    o,
		session: session{gwMAC: t.GatewayMAC()},
	}
}

// GatewayMAC returns the MAC address of the connected node.
func (c *Client) GatewayMAC() MACAddress {
	return c.session.gwMAC
}

// Login performs the pairing handshake, derives the session key, and starts
// the heartbeat loop. Must be called before any Send or Notifications.
func (c *Client) Login(ctx context.Context, meshName, meshPassword string) error {
	var randA [8]byte
	if _, err := rand.Read(randA[:]); err != nil {
		return fmt.Errorf("pigsydust: generating randA: %w", err)
	}

	salt, err := randomSalt()
	if err != nil {
		return err
	}

	req := crypto.BuildLoginRequest(meshName, meshPassword, randA)
	if err := c.t.WritePair(ctx, req[:]); err != nil {
		return fmt.Errorf("%w: writing pair: %w", ErrLoginFailed, err)
	}

	resp, err := c.t.ReadPair(ctx)
	if err != nil {
		return fmt.Errorf("%w: reading pair: %w", ErrLoginFailed, err)
	}
	randB, err := crypto.ParseLoginResponse(resp)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoginFailed, err)
	}

	sk := crypto.DeriveSessionKey(meshName, meshPassword, randA, randB)

	c.session.mu.Lock()
	c.session.key = sk
	c.session.salt = salt
	c.session.seq = 0
	c.session.loggedIn = true
	c.session.mu.Unlock()

	c.startHeartbeat()
	return nil
}

// Send encrypts cmd under the session key and writes it to CHAR_CMD. It's
// safe to call from multiple goroutines concurrently.
func (c *Client) Send(ctx context.Context, cmd command.Command) error {
	c.session.mu.Lock()
	if !c.session.loggedIn {
		c.session.mu.Unlock()
		return ErrNotLoggedIn
	}
	sk := c.session.key
	gwMAC := c.session.gwMAC
	c.session.mu.Unlock()

	sno := c.session.nextSNO()
	nonce := crypto.CommandNonce([6]byte(gwMAC), sno)
	packet := crypto.Encrypt(sk, nonce, sno, cmd.Encode())
	return c.t.WriteCommand(ctx, packet)
}

// Notifications subscribes to CHAR_NOTIFY and returns a channel of decrypted
// [Notification] values. The channel closes when ctx is cancelled or the
// underlying subscription ends.
//
// Notifications that fail CBC-MAC verification (stale packets from a prior
// session, mesh retransmits already consumed) are dropped and logged at
// debug level. Only one subscription per Client is supported.
func (c *Client) Notifications(ctx context.Context) (<-chan Notification, error) {
	c.notifyMu.Lock()
	if c.notifyActive {
		c.notifyMu.Unlock()
		return nil, errors.New("pigsydust: notifications already subscribed")
	}
	c.notifyActive = true
	c.notifyMu.Unlock()

	c.session.mu.Lock()
	if !c.session.loggedIn {
		c.session.mu.Unlock()
		c.notifyMu.Lock()
		c.notifyActive = false
		c.notifyMu.Unlock()
		return nil, ErrNotLoggedIn
	}
	sk := c.session.key
	gwMAC := c.session.gwMAC
	c.session.mu.Unlock()

	raw, err := c.t.SubscribeNotify(ctx)
	if err != nil {
		c.notifyMu.Lock()
		c.notifyActive = false
		c.notifyMu.Unlock()
		return nil, fmt.Errorf("pigsydust: subscribing notify: %w", err)
	}

	out := make(chan Notification, 64)
	go func() {
		defer close(out)
		defer func() {
			c.notifyMu.Lock()
			c.notifyActive = false
			c.notifyMu.Unlock()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case pkt, ok := <-raw:
				if !ok {
					return
				}
				n, err := DecryptNotification(sk, gwMAC, pkt)
				if err != nil {
					c.opts.logger.Debug("notification decrypt failed",
						"err", err, "len", len(pkt))
					continue
				}
				select {
				case out <- n:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// Close stops the heartbeat loop. The transport is not closed — the caller
// owns its lifecycle.
func (c *Client) Close(_ context.Context) error {
	if c.hbCancel != nil {
		c.hbCancel()
		<-c.hbDone
	}
	return nil
}

func (c *Client) startHeartbeat() {
	c.hbOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		c.hbCancel = cancel
		c.hbDone = make(chan struct{})
		go c.heartbeatLoop(ctx)
	})
}

func (c *Client) heartbeatLoop(ctx context.Context) {
	defer close(c.hbDone)
	// First heartbeat fires after one interval — the 30s Telink timer
	// resets on the login read, so there's headroom before the first tick.
	t := time.NewTicker(c.opts.heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := c.t.ReadPair(ctx); err != nil {
				c.opts.logger.Debug("heartbeat read failed", "err", err)
			}
		}
	}
}
