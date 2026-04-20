package pigsydust

import (
	"log/slog"
	"time"
)

// ClientOption configures a [Client]. Pass options to [NewClient].
type ClientOption func(*clientOptions)

type clientOptions struct {
	heartbeatInterval time.Duration
	logger            *slog.Logger
}

var defaultOptions = clientOptions{
	// Telink keepalive timer is 30s — stay under it comfortably.
	heartbeatInterval: 25 * time.Second,
	logger:            slog.Default(),
}

// WithHeartbeatInterval overrides the keepalive read interval. Values ≥ 30s
// risk the firmware tearing the connection down.
func WithHeartbeatInterval(d time.Duration) ClientOption {
	return func(o *clientOptions) { o.heartbeatInterval = d }
}

// WithLogger sets the slog logger used for session events (heartbeat, notify
// demux, decrypt failures). Defaults to [slog.Default].
func WithLogger(l *slog.Logger) ClientOption {
	return func(o *clientOptions) { o.logger = l }
}
