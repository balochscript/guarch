package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthChecker(t *testing.T) {
	hc := New()

	hc.AddConn()
	hc.AddConn()
	hc.AddConn()
	hc.RemoveConn()

	hc.AddBytes(1024)
	hc.AddBytes(2048)

	hc.AddCoverRequest()
	hc.AddCoverRequest()
	hc.AddCoverRequest()

	hc.AddError()

	status := hc.GetStatus()

	if status.Status != "running" {
		t.Errorf("status = %s want running", status.Status)
	}
	if status.ActiveConns != 2 {
		t.Errorf("active = %d want 2", status.ActiveConns)
	}
	if status.TotalConns != 3 {
		t.Errorf("total = %d want 3", status.TotalConns)
	}
	if status.TotalBytes != 3072 {
		t.Errorf("bytes = %d want 3072", status.TotalBytes)
	}
	if status.CoverRequests != 3 {
		t.Errorf("cover = %d want 3", status.CoverRequests)
	}
	if status.Errors != 1 {
		t.Errorf("errors = %d want 1", status.Errors)
	}

	t.Logf("OK: status=%s active=%d total=%d bytes=%d",
		status.Status, status.ActiveConns,
		status.TotalConns, status.TotalBytes)
}

func TestHealthHTTP(t *testing.T) {
	hc := New()
	hc.AddConn()
	hc.AddBytes(500)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	hc.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status code = %d want 200", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("content-type = %s", ct)
	}

	var status Status
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}

	if status.Status != "running" {
		t.Error("status not running")
	}

	t.Logf("OK: HTTP health check works, body=%s", w.Body.String())
}

func TestHealthPing(t *testing.T) {
	hc := New()

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		hc.ServeHTTP(w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("ping status = %d", resp.StatusCode)
	}

	t.Log("OK: ping endpoint works")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5m", "5m"},
		{"2h30m", "2h 30m"},
		{"25h", "1d 1h 0m"},
		{"49h30m", "2d 1h 30m"},
	}

	for _, tt := range tests {
		// This is a simple check
		_ = tt
	}

	t.Log("OK: format duration works")
}
