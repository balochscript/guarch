package cover

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestManagerWithMockServer(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "fake response for %s", r.URL.Path)
		},
	))
	defer server.Close()

	cfg := &Config{
		Enabled:       true,
		MaxConcurrent: 1,
		Domains: []DomainConfig{
			{
				Domain:      server.Listener.Addr().String(),
				Paths:       []string{"/page1", "/page2"},
				Weight:      100,
				MinInterval: 50 * time.Millisecond,
				MaxInterval: 100 * time.Millisecond,
			},
		},
	}

	m := NewManagerWithClient(cfg, server.Client(), nil)

	// ✅ فیکس: تایم‌اوت بیشتر — initDelay تا ۵ ثانیه طول میکشه
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m.Start(ctx)

	// ✅ فیکس: poll بجای sleep ثابت — صبر تا worker شروع کنه
	for i := 0; i < 80; i++ {
		if m.Stats().SampleCount() > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	stats := m.Stats()
	count := stats.SampleCount()
	if count == 0 {
		t.Error("expected some requests to be made")
	}

	avg := stats.AvgPacketSize()
	t.Logf("OK: %d requests, avg size=%d", count, avg)
}

func TestManagerSendOne(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello from mock"))
		},
	))
	defer server.Close()

	cfg := &Config{
		Enabled: true,
		Domains: []DomainConfig{
			{
				Domain:      server.Listener.Addr().String(),
				Paths:       []string{"/"},
				Weight:      100,
				MinInterval: time.Second,
				MaxInterval: 2 * time.Second,
			},
		},
	}

	m := NewManagerWithClient(cfg, server.Client(), nil)

	m.SendOne()

	count := m.Stats().SampleCount()
	if count != 1 {
		t.Errorf("count = %d want 1", count)
	}

	t.Logf("OK: single request sent, stats recorded")
}

func TestManagerPickDomain(t *testing.T) {
	cfg := &Config{
		Domains: []DomainConfig{
			{Domain: "heavy.com", Weight: 90, Paths: []string{"/"}},
			{Domain: "light.com", Weight: 10, Paths: []string{"/"}},
		},
	}

	m := NewManager(cfg, nil)

	heavyCount := 0
	for i := 0; i < 100; i++ {
		dc := m.pickDomain()
		if dc.Domain == "heavy.com" {
			heavyCount++
		}
	}

	if heavyCount < 60 {
		t.Errorf("heavy picked %d/100, expected more", heavyCount)
	}

	t.Logf("OK: heavy=%d/100 light=%d/100", heavyCount, 100-heavyCount)
}

func TestManagerNotRunning(t *testing.T) {
	m := NewManager(nil, nil)

	if m.IsRunning() {
		t.Error("should not be running before Start")
	}

	t.Log("OK: not running before start")
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Domains) == 0 {
		t.Error("default config has no domains")
	}

	for _, d := range cfg.Domains {
		if d.Domain == "" {
			t.Error("empty domain")
		}
		if len(d.Paths) == 0 {
			t.Error("no paths for", d.Domain)
		}
		if d.Weight <= 0 {
			t.Error("bad weight for", d.Domain)
		}
	}

	t.Logf("OK: default config has %d domains", len(cfg.Domains))
}
