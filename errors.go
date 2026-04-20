package pigsydust

import "errors"

// Errors returned by package pigsydust.
var (
	// ErrNotLoggedIn is returned by Client operations invoked before Login.
	ErrNotLoggedIn = errors.New("pigsydust: not logged in")

	// ErrLoginFailed is returned when the pairing handshake fails.
	ErrLoginFailed = errors.New("pigsydust: login failed")

	// ErrTagMismatch is returned when a notification's CBC-MAC tag fails
	// verification. This is common for stale packets from a prior session
	// and usually safe to ignore at the call site.
	ErrTagMismatch = errors.New("pigsydust: CBC-MAC tag mismatch")

	// ErrShortPacket is returned when a packet is shorter than the protocol
	// requires.
	ErrShortPacket = errors.New("pigsydust: packet too short")

	// ErrUnexpectedOpcode is returned when a parser is handed a notification
	// with the wrong opcode.
	ErrUnexpectedOpcode = errors.New("pigsydust: unexpected opcode")
)
