package piggsydust

import (
	"context"
	"sync"
	"sync/atomic"
)

// session holds per-connection state derived during login.
type session struct {
	sessionKey  [16]byte
	gwMAC       MACAddress
	sessionSalt [2]byte
	seqNum      atomic.Uint32

	// Notification routing.
	mu      sync.Mutex
	waiters map[waiterKey]chan Notification
}

type waiterKey struct {
	opcode byte
	source Address
}

// nextSNO returns the 3-byte sequence number for the next command
// and increments the counter.
func (s *session) nextSNO() [3]byte {
	seq := s.seqNum.Add(1) - 1
	return [3]byte{
		byte(seq),
		s.sessionSalt[0],
		s.sessionSalt[1],
	}
}

// registerWaiter creates a channel that will receive the next notification
// matching the given opcode from any source. Use source=0 to match any source.
func (s *session) registerWaiter(opcode byte, source Address) chan Notification {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan Notification, 1)
	if s.waiters == nil {
		s.waiters = make(map[waiterKey]chan Notification)
	}
	s.waiters[waiterKey{opcode: opcode, source: source}] = ch
	return ch
}

// unregisterWaiter removes a previously registered waiter.
func (s *session) unregisterWaiter(opcode byte, source Address) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.waiters, waiterKey{opcode: opcode, source: source})
}

// routeNotification checks if any waiter is expecting this notification.
// Returns true if the notification was consumed by a waiter.
func (s *session) routeNotification(n Notification) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try exact match first (opcode + source).
	key := waiterKey{opcode: n.Opcode, source: n.Source}
	if ch, ok := s.waiters[key]; ok {
		select {
		case ch <- n:
		default:
		}
		delete(s.waiters, key)
		return true
	}

	// Try wildcard source match (opcode + source=0).
	key = waiterKey{opcode: n.Opcode, source: 0}
	if ch, ok := s.waiters[key]; ok {
		select {
		case ch <- n:
		default:
		}
		delete(s.waiters, key)
		return true
	}

	return false
}

// waitForNotification registers a waiter and blocks until a matching
// notification arrives or the context is cancelled.
func (s *session) waitForNotification(ctx context.Context, opcode byte, source Address) (Notification, error) {
	ch := s.registerWaiter(opcode, source)
	defer s.unregisterWaiter(opcode, source)

	select {
	case n := <-ch:
		return n, nil
	case <-ctx.Done():
		return Notification{}, ctx.Err()
	}
}
