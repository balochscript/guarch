package cover

import (
	"testing"
	"time"
)

func TestShaperPadding(t *testing.T) {
	s := NewStats(10)
	s.Record(500)
	s.Record(500)
	s.Record(500)

	shaper := NewShaper(s, PatternWebBrowsing)

	padding := shaper.PaddingSize(100)
	if padding < 300 || padding > 500 {
		t.Errorf("padding = %d, expected around 400", padding)
	}

	padding2 := shaper.PaddingSize(600)
	if padding2 != 0 {
		t.Errorf("padding = %d, expected 0 for large data", padding2)
	}

	t.Logf("OK: padding for 100 bytes = %d", padding)
	t.Logf("OK: padding for 600 bytes = %d", padding2)
}

func TestShaperDelay(t *testing.T) {
	s := NewStats(10)
	s.Record(100)
	time.Sleep(10 * time.Millisecond)
	s.Record(100)

	shaper := NewShaper(s, PatternWebBrowsing)
	delay := shaper.Delay()

	if delay < 0 || delay > 30*time.Second {
		t.Errorf("delay = %v, seems wrong", delay)
	}

	t.Logf("OK: delay=%v", delay)
}

func TestShaperPatterns(t *testing.T) {
	s := NewStats(10)
	s.Record(500)
	s.Record(500)

	patterns := []struct {
		name    string
		pattern Pattern
	}{
		{"WebBrowsing", PatternWebBrowsing},
		{"VideoStream", PatternVideoStream},
		{"FileDownload", PatternFileDownload},
	}

	for _, p := range patterns {
		shaper := NewShaper(s, p.pattern)

		delay := shaper.Delay()
		idle := shaper.IdleDelay()
		frag := shaper.FragmentSize()
		pad := shaper.PaddingSize(100)

		t.Logf("OK: %s delay=%v idle=%v frag=%d pad=%d",
			p.name, delay, idle, frag, pad)
	}
}

func TestShaperFragment(t *testing.T) {
	s := NewStats(10)
	s.Record(1000)
	s.Record(1000)

	shaper := NewShaper(s, PatternWebBrowsing)

	frag := shaper.FragmentSize()
	if frag < 800 || frag > 1200 {
		t.Errorf("fragment = %d, expected around 1000", frag)
	}

	t.Logf("OK: fragment size=%d", frag)
}

func TestShaperShouldPad(t *testing.T) {
	s := NewStats(10)
	s.Record(500)

	shaper := NewShaper(s, PatternWebBrowsing)

	trueCount := 0
	for i := 0; i < 100; i++ {
		if shaper.ShouldSendPadding() {
			trueCount++
		}
	}

	if trueCount == 0 || trueCount == 100 {
		t.Errorf("shouldPad always %v, expected mix", trueCount == 100)
	}

	t.Logf("OK: shouldPad true %d/100 times", trueCount)
}
