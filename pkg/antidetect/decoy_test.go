package antidetect

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecoyHomePage(t *testing.T) {
	ds := NewDecoyServer()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	ds.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d want 200", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "CloudFront") {
		t.Error("missing CloudFront in body")
	}
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("content-type = %s", ct)
	}

	srv := w.Header().Get("Server")
	if srv != "nginx/1.24.0" {
		t.Errorf("server = %s want nginx", srv)
	}

	t.Logf("OK: home page %d bytes, server=%s", len(body), srv)
}

func TestDecoy404(t *testing.T) {
	ds := NewDecoyServer()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	ds.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d want 404", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "404") {
		t.Error("missing 404 in body")
	}

	t.Logf("OK: 404 page %d bytes", len(body))
}

func TestDecoyAllPages(t *testing.T) {
	ds := NewDecoyServer()

	pages := []struct {
		path   string
		status int
	}{
		{"/", 200},
		{"/about", 200},
		{"/contact", 200},
		{"/blog", 200},
		{"/unknown", 404},
		{"/admin", 404},
	}

	for _, p := range pages {
		req := httptest.NewRequest("GET", p.path, nil)
		w := httptest.NewRecorder()

		ds.ServeHTTP(w, req)

		if w.Code != p.status {
			t.Errorf("%s: status = %d want %d", p.path, w.Code, p.status)
		}

		t.Logf("OK: %s -> %d (%d bytes)", p.path, w.Code, w.Body.Len())
	}
}

func TestDecoyHeaders(t *testing.T) {
	ds := NewDecoyServer()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	ds.ServeHTTP(w, req)

	headers := []string{
		"Server",
		"Content-Type",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Cache-Control",
		"Date",
	}

	for _, h := range headers {
		val := w.Header().Get(h)
		if val == "" {
			t.Errorf("missing header: %s", h)
		} else {
			t.Logf("OK: %s = %s", h, val)
		}
	}
}

func TestDecoyAsHTTPServer(t *testing.T) {
	ds := NewDecoyServer()
	server := httptest.NewServer(ds)
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d want 200", resp.StatusCode)
	}

	t.Logf("OK: decoy server works at %s", server.URL)
}
