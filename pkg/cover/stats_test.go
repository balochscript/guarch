package cover

import (
	"testing"
	"time"
)

func TestStatsRecord(t *testing.T) {
	s := NewStats(10)

	s.Record(100)
	s.Record(200)
	s.Record(300)

	avg := s.AvgPacketSize()
	if avg != 200 {
		t.Errorf("avg = %d want 200", avg)
	}

	count := s.SampleCount()
	if count != 3 {
		t.Errorf("count = %d want 3", count)
	}

	t.Logf("OK: avg=%d count=%d", avg, count)
}

func TestStatsEmpty(t *testing.T) {
	s := NewStats(10)

	avg := s.AvgPacketSize()
	if avg != 512 {
		t.Errorf("empty avg = %d want 512", avg)
	}

	interval := s.AvgInterval()
	if interval != 2*time.Second {
		t.Errorf("empty interval = %v want 2s", interval)
	}

	t.Logf("OK: defaults work for empty stats")
}

func TestStatsMinMax(t *testing.T) {
	s := NewStats(10)

	s.Record(100)
	s.Record(500)
	s.Record(200)
	s.Record(800)
	s.Record(50)

	min, max := s.MinMaxPacketSize()
	if min != 50 {
		t.Errorf("min = %d want 50", min)
	}
	if max != 800 {
		t.Errorf("max = %d want 800", max)
	}

	t.Logf("OK: min=%d max=%d", min, max)
}

func TestStatsOverflow(t *testing.T) {
	s := NewStats(5)

	for i := 0; i < 20; i++ {
		s.Record(i * 100)
	}

	count := s.SampleCount()
	if count != 5 {
		t.Errorf("count = %d want 5 (max samples)", count)
	}

	t.Logf("OK: overflow handled, count=%d", count)
}

func TestStatsTotalSent(t *testing.T) {
	s := NewStats(10)

	s.Record(100)
	s.Record(200)
	s.Record(300)

	total := s.TotalSent()
	if total != 3 {
		t.Errorf("total = %d want 3", total)
	}

	t.Logf("OK: total sent=%d", total)
}

func TestStatsInterval(t *testing.T) {
	s := NewStats(10)

	s.Record(100)
	time.Sleep(50 * time.Millisecond)
	s.Record(200)
	time.Sleep(50 * time.Millisecond)
	s.Record(300)

	interval := s.AvgInterval()
	if interval < 30*time.Millisecond || interval > 200*time.Millisecond {
		t.Errorf("interval = %v, seems wrong", interval)
	}

	t.Logf("OK: avg interval=%v", interval)
}
