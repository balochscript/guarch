package cover

import (
	"crypto/rand"
	"math/big"
)

type SmartPadder struct {
	maxPadding int
	adaptive   *AdaptiveCover
}

// Web bucket sizes based on real HTTPS traffic analysis
var webBuckets = []int{
	64,    // Small ACKs
	128,   // Small responses
	256,   // JSON/HTML fragments
	512,   // Medium responses
	1024,  // HTML pages
	1460,  // MTU-optimized
	2048,  // Images/fonts
	4096,  // Large JSON
	8192,  // JS bundles
	16384, // Large resources
}

func NewSmartPadder(maxPadding int, adaptive *AdaptiveCover) *SmartPadder {
	return &SmartPadder{
		maxPadding: maxPadding,
		adaptive:   adaptive,
	}
}

// Calculate returns padding size to reach nearest web bucket with ±10% jitter
func (sp *SmartPadder) Calculate(payloadSize int) int {
	maxPad := sp.maxPadding
	if sp.adaptive != nil {
		maxPad = sp.adaptive.GetMaxPadding()
	}

	if maxPad <= 0 {
		return 0
	}

	// Find target bucket
	targetSize := payloadSize
	for _, b := range webBuckets {
		if payloadSize <= b {
			targetSize = b
			break
		}
	}

	// Round up to MTU if larger than biggest bucket
	if targetSize <= payloadSize {
		targetSize = ((payloadSize / 1460) + 1) * 1460
	}

	padding := targetSize - payloadSize

	// Clamp to maxPadding
	if padding > maxPad {
		padding = maxPad
	}

	// Apply ±10% jitter (only if padding > 20)
	if padding > 20 {
		jitterMax := padding / 10
		if jitterMax > 0 {
			jitter := secureRandInt(jitterMax)
			
			// Apply jitter only if it doesn't overflow/underflow
			if secureRandBool() {
				if padding+jitter <= maxPad {
					padding += jitter
				}
			} else {
				if padding-jitter >= 0 {
					padding -= jitter
				}
			}
		}
	}

	// Final safety clamps
	if padding < 0 {
		padding = 0
	}
	if padding > maxPad {
		padding = maxPad
	}

	return padding
}

// secureRandInt returns cryptographically secure random int in [0, max)
func secureRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0 // fallback to no randomness if crypto/rand fails
	}
	return int(n.Int64())
}

// secureRandBool returns cryptographically secure random boolean
func secureRandBool() bool {
	return secureRandInt(2) == 0
}
