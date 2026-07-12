package sni

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand"
	"sync/atomic"
	"time"
)

type Selector struct {
	mode         SelectionMode
	domains      []Domain
	currentIndex atomic.Uint32
}

func NewSelector(mode SelectionMode, domains []Domain) *Selector {
	if len(domains) == 0 {
		domains = []Domain{}
	}
	return &Selector{
		mode:    mode,
		domains: domains,
	}
}

func (s *Selector) Select() (string, error) {
	return s.SelectFrom(s.domains)
}

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

func (s *Selector) selectRandom(domains []Domain) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("no domains")
	}
	if len(domains) == 1 {
		return domains[0].Domain, nil
	}
	idx := cryptoRandInt(len(domains))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(domains) {
		idx = len(domains) - 1
	}
	return domains[idx].Domain, nil
}

func (s *Selector) selectWeighted(domains []Domain) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("no domains")
	}
	if len(domains) == 1 {
		return domains[0].Domain, nil
	}
	totalWeight := 0
	for _, d := range domains {
		if d.Weight > 0 {
			totalWeight += d.Weight
		}
	}
	if totalWeight <= 0 {
		return s.selectRandom(domains)
	}
	r := cryptoRandInt(totalWeight)
	if r < 0 {
		r = 0
	}
	for _, d := range domains {
		if d.Weight <= 0 {
			continue
		}
		r -= d.Weight
		if r < 0 {
			return d.Domain, nil
		}
	}
	return domains[0].Domain, nil
}

func (s *Selector) selectSequential(domains []Domain) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("no domains")
	}
	if len(domains) == 1 {
		return domains[0].Domain, nil
	}
	idx := s.currentIndex.Add(1) % uint32(len(domains))
	return domains[idx].Domain, nil
}

func cryptoRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	if max == 1 {
		return 0
	}
	if max > 1000000 {
		max = 1000000
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		mrand.Seed(time.Now().UnixNano())
		return mrand.Intn(max)
	}
	return int(n.Int64())
}
