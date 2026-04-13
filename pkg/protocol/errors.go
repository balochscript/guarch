package protocol

import (
	"errors"
	"fmt"
)

// Base Protocol Errors
var (
	ErrPacketTooShort    = errors.New("guarch: packet too short")
	ErrPacketTooLarge    = errors.New("guarch: packet exceeds max size")
	ErrInvalidVersion    = errors.New("guarch: invalid protocol version")
	ErrInvalidPacketType = errors.New("guarch: invalid packet type")

	ErrAuthFailed  = errors.New("guarch: authentication failed")
	ErrAuthTimeout = errors.New("guarch: authentication timeout")

	ErrDecryptFailed    = errors.New("guarch: decryption failed")
	ErrConnectionClosed = errors.New("guarch: connection closed")
	ErrReplayDetected   = errors.New("guarch: replay detected")
	ErrMuxClosed        = errors.New("guarch: mux closed")
)

// 🆕 NEW: Cryptography Errors
var (
	// Key Exchange Errors
	ErrInvalidPublicKey    = errors.New("guarch/crypto: invalid public key")
	ErrLowOrderPoint       = errors.New("guarch/crypto: low-order point detected")
	ErrKeyExchangeFailed   = errors.New("guarch/crypto: key exchange failed")
	ErrSharedSecretZero    = errors.New("guarch/crypto: shared secret is zero")
	
	// Encryption Errors
	ErrKeyExhausted        = errors.New("guarch/crypto: key exhausted - rotation required")
	ErrInvalidKeySize      = errors.New("guarch/crypto: invalid key size")
	ErrNonceGeneration     = errors.New("guarch/crypto: nonce generation failed")
	ErrEncryptionFailed    = errors.New("guarch/crypto: encryption failed")
	
	// AEAD Errors
	ErrAADMismatch         = errors.New("guarch/crypto: AAD mismatch")
	ErrAuthTagInvalid      = errors.New("guarch/crypto: authentication tag invalid")
)

// 🆕 NEW: Connection Errors
var (
	ErrHandshakeFailed     = errors.New("guarch/conn: handshake failed")
	ErrPSKMismatch         = errors.New("guarch/conn: PSK mismatch")
	ErrCertPinMismatch     = errors.New("guarch/conn: certificate pin mismatch")
	ErrTimeoutReached      = errors.New("guarch/conn: timeout reached")
	ErrConnectionReset     = errors.New("guarch/conn: connection reset by peer")
	ErrTooManyRetries      = errors.New("guarch/conn: too many retries")
)

// 🆕 NEW: Replay Protection Errors
var (
	ErrDuplicatePacket     = errors.New("guarch/replay: duplicate packet")
	ErrPacketTooOld        = errors.New("guarch/replay: packet too old")
	ErrSequenceOutOfOrder  = errors.New("guarch/replay: sequence out of order")
)

// 🆕 NEW: Stream/Mux Errors
var (
	ErrStreamNotFound      = errors.New("guarch/mux: stream not found")
	ErrStreamClosed        = errors.New("guarch/mux: stream closed")
	ErrStreamLimitReached  = errors.New("guarch/mux: stream limit reached")
	ErrBufferOverflow      = errors.New("guarch/mux: buffer overflow")
)

// 🆕 NEW: SOCKS5 Errors
var (
	ErrSOCKSVersionMismatch = errors.New("guarch/socks5: version mismatch")
	ErrSOCKSAuthFailed      = errors.New("guarch/socks5: authentication failed")
	ErrSOCKSCommandNotSupported = errors.New("guarch/socks5: command not supported")
	ErrSOCKSAddressTypeInvalid  = errors.New("guarch/socks5: invalid address type")
	ErrSOCKSConnectionRefused   = errors.New("guarch/socks5: connection refused")
)

// 🆕 NEW: Cover Traffic Errors
var (
	ErrCoverTrafficFailed  = errors.New("guarch/cover: cover traffic request failed")
	ErrDomainUnreachable   = errors.New("guarch/cover: domain unreachable")
	ErrRateLimitExceeded   = errors.New("guarch/cover: rate limit exceeded")
)

// 🆕 NEW: DNS Tunnel Errors (برای survival mode)
var (
	ErrDNSQueryFailed      = errors.New("guarch/dns: query failed")
	ErrDNSEncodeFailed     = errors.New("guarch/dns: encode failed")
	ErrDNSDecodeFailed     = errors.New("guarch/dns: decode failed")
	ErrDNSResponseInvalid  = errors.New("guarch/dns: invalid response")
	ErrDNSTunnelClosed     = errors.New("guarch/dns: tunnel closed")
)

// 🆕 NEW: Helper functions for error wrapping

// WrapError wraps an error with context
func WrapError(base error, context string) error {
	if base == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, base)
}

// IsTemporary checks if an error is temporary and can be retried
func IsTemporary(err error) bool {
	if err == nil {
		return false
	}
	
	// Temporary errors که retry کردنشون منطقیه
	temporary := []error{
		ErrAuthTimeout,
		ErrTimeoutReached,
		ErrConnectionReset,
		ErrCoverTrafficFailed,
		ErrDomainUnreachable,
		ErrDNSQueryFailed,
	}
	
	for _, tempErr := range temporary {
		if errors.Is(err, tempErr) {
			return true
		}
	}
	
	return false
}

// IsFatal checks if an error is fatal and connection should be closed
func IsFatal(err error) bool {
	if err == nil {
		return false
	}
	
	// Fatal errors که نباید retry کرد
	fatal := []error{
		ErrAuthFailed,
		ErrPSKMismatch,
		ErrCertPinMismatch,
		ErrInvalidVersion,
		ErrKeyExhausted,
		ErrLowOrderPoint,
		ErrInvalidPublicKey,
	}
	
	for _, fatalErr := range fatal {
		if errors.Is(err, fatalErr) {
			return true
		}
	}
	
	return false
}

// IsCryptoError checks if error is cryptography-related
func IsCryptoError(err error) bool {
	if err == nil {
		return false
	}
	
	cryptoErrors := []error{
		ErrInvalidPublicKey,
		ErrLowOrderPoint,
		ErrKeyExchangeFailed,
		ErrSharedSecretZero,
		ErrKeyExhausted,
		ErrInvalidKeySize,
		ErrNonceGeneration,
		ErrEncryptionFailed,
		ErrDecryptFailed,
		ErrAADMismatch,
		ErrAuthTagInvalid,
	}
	
	for _, cryptoErr := range cryptoErrors {
		if errors.Is(err, cryptoErr) {
			return true
		}
	}
	
	return false
}

// IsReplayError checks if error is replay attack related
func IsReplayError(err error) bool {
	if err == nil {
		return false
	}
	
	replayErrors := []error{
		ErrReplayDetected,
		ErrDuplicatePacket,
		ErrPacketTooOld,
		ErrSequenceOutOfOrder,
	}
	
	for _, replayErr := range replayErrors {
		if errors.Is(err, replayErr) {
			return true
		}
	}
	
	return false
}

// 🆕 NEW: Error Details for logging/debugging
type ErrorDetails struct {
	Code      string
	Message   string
	IsFatal   bool
	IsRetryable bool
	Category  string
}

// GetErrorDetails returns detailed information about an error
func GetErrorDetails(err error) ErrorDetails {
	if err == nil {
		return ErrorDetails{
			Code:    "OK",
			Message: "no error",
		}
	}
	
	details := ErrorDetails{
		Message:     err.Error(),
		IsFatal:     IsFatal(err),
		IsRetryable: IsTemporary(err),
	}
	
	// Categorize error
	switch {
	case IsCryptoError(err):
		details.Category = "crypto"
		details.Code = "CRYPTO_ERROR"
	case IsReplayError(err):
		details.Category = "replay"
		details.Code = "REPLAY_ATTACK"
	case errors.Is(err, ErrConnectionClosed):
		details.Category = "connection"
		details.Code = "CONN_CLOSED"
	case errors.Is(err, ErrAuthFailed):
		details.Category = "auth"
		details.Code = "AUTH_FAILED"
	case errors.Is(err, ErrMuxClosed):
		details.Category = "mux"
		details.Code = "MUX_CLOSED"
	default:
		details.Category = "unknown"
		details.Code = "UNKNOWN_ERROR"
	}
	
	return details
}
