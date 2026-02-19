package antidetect

import (
	"fmt"
	"testing"
	"time"
)

func TestProbeNormal(t *testing.T) {
	pd := NewProbeDetector(10, time.Minute)

	suspicious := pd.Check("192.168.1.1:1234")
	if suspicious {
		t.Error("single request should not be suspicious")
	}

	t.Log("OK: single request is normal")
}

func TestProbeSuspicious(t *testing.T) {
	pd := NewProbeDetector(5, time.Minute)

	for i := 0; i < 6; i++ {
		suspicious := pd.Check("10.0.0.1:1234")
		if i < 5 && suspicious {
			t.Errorf("request %d should not be suspicious", i)
		}
		if i >= 5 && !suspicious {
			t.Errorf("request %d should be suspicious", i)
		}
	}

	t.Log("OK: detected suspicious behavior after 5 attempts")
}

func TestProbeDifferentIPs(t *testing.T) {
	pd := NewProbeDetector(3, time.Minute)

	for i := 0; i < 5; i++ {
		addr := fmt.Sprintf("192.168.1.%d:1234", i)
		suspicious := pd.Check(addr)
		if suspicious {
			t.Errorf("IP %s should not be suspicious (only 1 attempt)", addr)
		}
	}

	t.Log("OK: different IPs tracked separately")
}

func TestProbeAttemptCount(t *testing.T) {
	pd := NewProbeDetector(10, time.Minute)

	pd.Check("1.2.3.4:80")
	pd.Check("1.2.3.4:80")
	pd.Check("1.2.3.4:80")

	count := pd.AttemptCount("1.2.3.4:80")
	if count != 3 {
		t.Errorf("count = %d want 3", count)
	}

	count2 := pd.AttemptCount("5.6.7.8:80")
	if count2 != 0 {
		t.Errorf("unknown IP count = %d want 0", count2)
	}

	t.Logf("OK: attempt counts correct")
}

func TestProbeWithPort(t *testing.T) {
	pd := NewProbeDetector(3, time.Minute)

	pd.Check("1.2.3.4:80")
	pd.Check("1.2.3.4:443")
	pd.Check("1.2.3.4:8080")

	count := pd.AttemptCount("1.2.3.4:80")
	if count != 3 {
		t.Errorf("count = %d want 3 (same IP different ports)", count)
	}

	t.Log("OK: same IP different ports counted together")
}
