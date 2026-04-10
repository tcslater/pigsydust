package crypto

import "errors"

var (
	// ErrLoginFailed indicates the login handshake did not produce a
	// valid response (wrong tag byte, short response, or auth mismatch).
	ErrLoginFailed = errors.New("piggsydust/crypto: login failed")

	// ErrTagMismatch indicates the CBC-MAC tag did not match during
	// notification decryption.
	ErrTagMismatch = errors.New("piggsydust/crypto: CBC-MAC tag mismatch")
)
