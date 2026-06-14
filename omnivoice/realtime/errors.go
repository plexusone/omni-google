package realtime

import "errors"

// Common errors.
var (
	// ErrSessionClosed is returned when operating on a closed session.
	ErrSessionClosed = errors.New("session closed")

	// ErrConnectionFailed is returned when WebSocket connection fails.
	ErrConnectionFailed = errors.New("websocket connection failed")

	// ErrSendFailed is returned when sending a message fails.
	ErrSendFailed = errors.New("failed to send message")

	// ErrAPIKeyRequired is returned when API key is not provided.
	ErrAPIKeyRequired = errors.New("API key is required")

	// ErrSetupFailed is returned when session setup fails.
	ErrSetupFailed = errors.New("session setup failed")
)
