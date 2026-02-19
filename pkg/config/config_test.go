package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()

	if cfg.Listen == "" {
		t.Error("empty listen")
	}
	if cfg.Server == "" {
		t.Error("empty server")
	}
	if !cfg.Cover.Enabled {
		t.Error("cover should be enabled")
	}
	if len(cfg.Cover.Domains) == 0 {
		t.Error("no cover domains")
	}

	t.Logf("OK: listen=%s server=%s domains=%d",
		cfg.Listen, cfg.Server, len(cfg.Cover.Domains))
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	if cfg.Listen == "" {
		t.Error("empty listen")
	}
	if cfg.DecoyAddr == "" {
		t.Error("empty decoy addr")
	}

	t.Logf("OK: listen=%s decoy=%s", cfg.Listen, cfg.DecoyAddr)
}

func TestSaveLoadClient(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "client.json")

	original := DefaultClientConfig()
	original.Server = "1.2.3.4:8443"

	if err := original.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadClient(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Server != original.Server {
		t.Errorf("server: got %s want %s", loaded.Server, original.Server)
	}
	if loaded.Listen != original.Listen {
		t.Errorf("listen: got %s want %s", loaded.Listen, original.Listen)
	}

	t.Log("OK: save and load client config works")
}

func TestSaveLoadServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.json")

	original := DefaultServerConfig()

	if err := original.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadServer(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Listen != original.Listen {
		t.Errorf("listen: got %s want %s", loaded.Listen, original.Listen)
	}

	t.Log("OK: save and load server config works")
}

func TestClientConfigJSON(t *testing.T) {
	cfg := DefaultClientConfig()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Client config JSON:\n%s", string(data))

	var parsed ClientConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	t.Log("OK: JSON roundtrip works")
}

func TestMissingServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	os.WriteFile(path, []byte(`{"listen":"127.0.0.1:1080"}`), 0644)

	_, err := LoadClient(path)
	if err == nil {
		t.Error("should fail without server address")
	}

	t.Logf("OK: missing server rejected: %v", err)
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1s", "1s"},
		{"5m", "5m0s"},
		{"100ms", "100ms"},
		{"bad", "5s"},
	}

	for _, tt := range tests {
		d := ParseDuration(tt.input)
		if d.String() != tt.expected {
			t.Errorf("ParseDuration(%q) = %v want %v",
				tt.input, d, tt.expected)
		}
	}

	t.Log("OK: duration parsing works")
}

func TestFileNotFound(t *testing.T) {
	_, err := LoadClient("/nonexistent/file.json")
	if err == nil {
		t.Error("should fail for missing file")
	}

	t.Logf("OK: missing file rejected: %v", err)
}
