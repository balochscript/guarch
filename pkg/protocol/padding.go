package protocol

import (
	"crypto/rand"
	"math/big"
)

// ═══════════════════════════════════════════════════════════
// Smart Padding Calculator
// ═══════════════════════════════════════════════════════════

// Web-realistic packet size buckets (bytes)
var webBuckets = []int{
	64,    // Tiny (ACK, keepalive)
	128,   // Small (DNS, API responses)
	256,   // Small-medium (JSON API)
	512,   // Medium (HTML snippets, small images)
	1024,  // Large (full HTML pages)
	1460,  // MTU-sized (TCP full packet)
	2048,  // Very large (images, scripts)
	4096,  // Extra large (bundled JS/CSS)
	8192,  // Huge (large images)
	16384, // Maximum (large assets)
}

// CalculateSmartPadding محاسبه padding برای رساندن به bucket نزدیک
func CalculateSmartPadding(payloadSize, maxPadding int) int {
	if maxPadding <= 0 {
		return 0
	}

	// پیدا کردن نزدیک‌ترین bucket بزرگتر
	targetSize := payloadSize
	for _, b := range webBuckets {
		if payloadSize <= b {
			targetSize = b
			break
		}
	}

	// اگر از بزرگترین bucket هم بزرگتر بود
	if targetSize <= payloadSize {
		// گرد کردن به نزدیک‌ترین MTU multiple
		targetSize = ((payloadSize / 1460) + 1) * 1460
	}

	padding := targetSize - payloadSize

	// محدود کردن به maxPadding
	if padding > maxPadding {
		padding = maxPadding
	}

	// اضافه کردن jitter ±10% برای جلوگیری از exact bucket sizes
	if padding > 20 {
		jitterMax := padding / 10
		if jitterMax > 0 {
			jitter := cryptoRandInt(jitterMax)
			if cryptoRandBool() {
				padding += jitter
			} else {
				padding -= jitter
			}
		}
	}

	// اطمینان از مثبت بودن
	if padding < 0 {
		padding = 0
	}

	// اطمینان نهایی از عدم تجاوز
	if padding > maxPadding {
		padding = maxPadding
	}

	return padding
}

// cryptoRandInt تولید عدد تصادفی امن (0 تا max-1)
func cryptoRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

// cryptoRandBool تولید boolean تصادفی امن
func cryptoRandBool() bool {
	return cryptoRandInt(2) == 0
}
