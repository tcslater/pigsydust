package pigsydust

import (
	"crypto/rand"
	"fmt"
	"sync"
)

// session holds the per-connection cryptographic state: the derived session
// key, the 2-byte salt (high bytes of every sno), and the monotonically
// incrementing sno counter.
type session struct {
	mu         sync.Mutex
	key        [16]byte
	salt       [2]byte
	seq        uint8
	gwMAC      MACAddress
	loggedIn   bool
}

// nextSNO returns the next 3-byte serial number to use for a command. The
// low byte is the monotonic counter; the two high bytes are the random salt
// chosen at login. Counter wraps naturally at 256.
func (s *session) nextSNO() [3]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	sno := [3]byte{s.seq, s.salt[0], s.salt[1]}
	s.seq++
	return sno
}

// randomSalt generates a 2-byte random salt for the session.
func randomSalt() ([2]byte, error) {
	var out [2]byte
	if _, err := rand.Read(out[:]); err != nil {
		return out, fmt.Errorf("pigsydust: generating salt: %w", err)
	}
	return out, nil
}
