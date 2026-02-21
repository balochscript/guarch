package protocol

import "errors"

var (
	ErrPacketTooShort    = errors.New("guarch: packet too short")
	ErrPacketTooLarge    = errors.New("guarch: packet exceeds max size")
	ErrInvalidVersion    = errors.New("guarch: invalid protocol version")
	ErrInvalidPacketType = errors.New("guarch: invalid packet type")

	ErrAuthFailed       = errors.New("guarch: authentication failed")
	ErrAuthTimeout      = errors.New("guarch: authentication timeout")

	ErrDecryptFailed    = errors.New("guarch: decryption failed")
	ErrConnectionClosed = errors.New("guarch: connection closed")
	ErrReplayDetected   = errors.New("guarch: replay detected")
	ErrMuxClosed        = errors.New("guarch: mux closed")
)
