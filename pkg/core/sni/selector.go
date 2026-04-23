package sni

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync/atomic"
)

// ═══════════════════════════════════════════════════════════
// Selector - انتخاب SNI بر اساس strategy
// ═══════════════════════════════════════════════════════════

// Selector انتخاب‌گر SNI
type Selector struct {
	mode    SelectionMode
	domains []Domain
	
	// برای sequential mode
	currentIndex atomic.Uint32
}

// NewSelector ساخت selector جدید
func NewSelector(mode SelectionMode, domains []Domain) *Selector {
	return &Selector{
		mode:    mode,
		domains: domains,
	}
}

// Select انتخاب یک SNI
func (s *Selector) Select() (string, error) {
	return s.SelectFrom(s.domains)
}

// SelectFrom انتخاب از لیست مشخص
func (s *Selector) SelectFrom(domains []Domain) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("no domains available")
	}
	
	switch s.mode {
	case ModeRandom:
		return s.selectRandom(domains)
		
	case ModeWeighted:
		return s.selectWeighted(domains)
		
	case ModeSequential:
		return s.selectSequential(domains)
		
	case ModeSingle:
		return domains[0].Domain, nil
		
	default:
		return s.selectRandom(domains)
	}
}

// selectRandom انتخاب تصادفی
func (s *Selector) selectRandom(domains []Domain) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("no domains")
	}
	
	idx := cryptoRandInt(len(domains))
	return domains[idx].Domain, nil
}

// selectWeighted انتخاب بر اساس وزن
func (s *Selector) selectWeighted(domains []Domain) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("no domains")
	}
	
	// محاسبه مجموع وزن‌ها
	totalWeight := 0
	for _, d := range domains {
		totalWeight += d.Weight
	}
	
	if totalWeight == 0 {
		// اگه همه وزن صفر بودن، random انتخاب کن
		return s.selectRandom(domains)
	}
	
	// انتخاب تصادفی بر اساس وزن
	r := cryptoRandInt(totalWeight)
	
	for _, d := range domains {
		r -= d.Weight
		if r < 0 {
			return d.Domain, nil
		}
	}
	
	// fallback (نباید برسیم اینجا)
	return domains[0].Domain, nil
}

// selectSequential انتخاب به ترتیب
func (s *Selector) selectSequential(domains []Domain) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("no domains")
	}
	
	idx := s.currentIndex.Add(1) % uint32(len(domains))
	return domains[idx].Domain, nil
}

// ═══════════════════════════════════════════════════════════
// Helper: Crypto-secure random
// ═══════════════════════════════════════════════════════════

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
