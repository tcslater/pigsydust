package pigsydust

import (
	"io"
	"log/slog"
	"time"
)

type clientConfig struct {
	heartbeatInterval time.Duration
	commandTimeout    time.Duration
	logger            *slog.Logger
	randSource        io.Reader
}

func defaultConfig() clientConfig {
	return clientConfig{
		heartbeatInterval: 30 * time.Second,
		commandTimeout:    5 * time.Second,
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// Option configures a [Client].
type Option func(*clientConfig)

// WithHeartbeatInterval sets the interval for keepalive reads on CHAR_PAIR.
// Default is 30 seconds.
func WithHeartbeatInterval(d time.Duration) Option {
	return func(c *clientConfig) { c.heartbeatInterval = d }
}

// WithCommandTimeout sets the timeout for request-response operations
// (status polls, group queries, LED queries, etc.). Default is 5 seconds.
func WithCommandTimeout(d time.Duration) Option {
	return func(c *clientConfig) { c.commandTimeout = d }
}

// WithLogger sets the structured logger for debug and error messages.
// Default discards all log output.
func WithLogger(l *slog.Logger) Option {
	return func(c *clientConfig) { c.logger = l }
}

// WithRandSource overrides the random byte source used for login nonce
// generation. Default is crypto/rand.Reader. This option is primarily
// useful for deterministic testing.
func WithRandSource(r io.Reader) Option {
	return func(c *clientConfig) { c.randSource = r }
}
