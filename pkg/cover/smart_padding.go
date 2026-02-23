package cover

import (
	"crypto/rand"
	"math/big"
)

// SmartPadder — padding هوشمند بر اساس bucket sizes
// هدف: بسته‌ها به اندازه‌های رایج وب گرد بشن
type SmartPadder struct {
	maxPadding int
	adaptive   *AdaptiveCover // nil = بدون adaptive
}

// اندازه‌های رایج بسته‌های وب (بایت)
var webBuckets = []int{
	64,    // TCP ACK, کوچک‌ترین
	128,   // DNS response
	256,   // API response کوچک
	512,   // JSON response
	1024,  // HTML fragment
	1460,  // TCP MSS (رایج‌ترین)
	2048,  // HTML page کوچک
	4096,  // HTML page متوسط
	8192,  // تصویر کوچک
	16384, // chunk ویدئو
}

func NewSmartPadder(maxPadding int, adaptive *AdaptiveCover) *SmartPadder {
	return &SmartPadder{
		maxPadding: maxPadding,
		adaptive:   adaptive,
	}
}

// Calculate — محاسبه‌ی padding برای یک بسته
func (sp *SmartPadder) Calculate(payloadSize int) int {
	// اگه adaptive فعاله، حداکثر padding رو از اون بگیر
	maxPad := sp.maxPadding
	if sp.adaptive != nil {
		maxPad = sp.adaptive.GetMaxPadding()
	}

	if maxPad <= 0 {
		return 0
	}

	// پیدا کردن نزدیک‌ترین bucket بزرگ‌تر
	targetSize := payloadSize
	for _, b := range webBuckets {
		if payloadSize <= b {
			targetSize = b
			break
		}
	}

	// اگه payload از همه‌ی bucket‌ها بزرگ‌تره
	if targetSize <= payloadSize {
		// گرد کردن به بالا به ضریب ۱۴۶۰ (TCP MSS)
		targetSize = ((payloadSize / 1460) + 1) * 1460
	}

	padding := targetSize - payloadSize

	// محدود به حداکثر
	if padding > maxPad {
		padding = maxPad
	}

	// اضافه کردن jitter ±۱۰% تا دقیقاً bucket size نباشه
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

	// حداقل صفر
	if padding < 0 {
		padding = 0
	}

	// دوباره محدود به حداکثر (بعد از jitter)
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
