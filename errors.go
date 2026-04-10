package piggsydust

import (
	"errors"
	"fmt"
)

var (
	// ErrNotConnected indicates the client has not completed login.
	ErrNotConnected = errors.New("piggsydust: not connected")

	// ErrClosed indicates the client has been closed.
	ErrClosed = errors.New("piggsydust: client closed")

	// ErrTimeout indicates an operation did not receive a response in time.
	ErrTimeout = errors.New("piggsydust: operation timed out")

	// ErrInvalidPacket indicates a malformed packet was received.
	ErrInvalidPacket = errors.New("piggsydust: invalid packet")

	// ErrInvalidChecksum indicates an alarm record XOR checksum mismatch.
	ErrInvalidChecksum = errors.New("piggsydust: invalid XOR checksum")
)

// OpError wraps an error with the operation and target address that caused it.
type OpError struct {
	Op   string
	Addr Address
	Err  error
}

func (e *OpError) Error() string {
	return fmt.Sprintf("piggsydust: %s %s: %v", e.Op, e.Addr, e.Err)
}

func (e *OpError) Unwrap() error {
	return e.Err
}
