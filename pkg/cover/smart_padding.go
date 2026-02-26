package cover

import (
	"crypto/rand"
	"math/big"
)

type SmartPadder struct {
	maxPadding int
	adaptive   *AdaptiveCover
}

var webBuckets = []int{
	64,
	128,
	256,
	512,
	1024,
	1460,
	2048,
	4096,
	8192,
	16384,
}

func NewSmartPadder(maxPadding int, adaptive *AdaptiveCover) *SmartPadder {
	return &SmartPadder{
		maxPadding: maxPadding,
		adaptive:   adaptive,
	}
}

func (sp *SmartPadder) Calculate(payloadSize int) int {
	maxPad := sp.maxPadding
	if sp.adaptive != nil {
		maxPad = sp.adaptive.GetMaxPadding()
	}

	if maxPad <= 0 {
		return 0
	}

	targetSize := payloadSize
	for _, b := range webBuckets {
		if payloadSize <= b {
			targetSize = b
			break
		}
	}

	if targetSize <= payloadSize {
		targetSize = ((payloadSize / 1460) + 1) * 1460
	}

	padding := targetSize - payloadSize

	if padding > maxPad {
		padding = maxPad
	}

	if padding > 20 {
		jitterMax := padding / 10
		if jitterMax > 0 {
			jitter := smartRandInt(jitterMax)
			if smartRandBool() {
				padding += jitter
			} else {
				padding -= jitter
			}
		}
	}

	if padding < 0 {
		padding = 0
	}

	if padding > maxPad {
		padding = maxPad
	}

	return padding
}

func smartRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func smartRandBool() bool {
	return smartRandInt(2) == 0
}
