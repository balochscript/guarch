package mobile

import (
	"encoding/json"
	"testing"
)

type testCallback struct {
	statuses []string
	logs     []string
}

func (tc *testCallback) OnStatusChanged(s string) { tc.statuses = append(tc.statuses, s) }
func (tc *testCallback) OnStatsUpdate(s string)   {}
func (tc *testCallback) OnLog(s string)            { tc.logs = append(tc.logs, s) }

func TestNewEngine(t *testing.T) {
	cb := &testCallback{}
	engine := NewEngine(cb)

	if engine.GetStatus() != "disconnected" {
		t.Errorf("status = %s want disconnected", engine.GetStatus())
	}

	t.Log("OK: engine created")
}

func TestGetStats(t *testing.T) {
	cb := &testCallback{}
	engine := NewEngine(cb)

	statsJson := engine.GetStats()

	var stats StatsData
	if err := json.Unmarshal([]byte(statsJson), &stats); err != nil {
		t.Fatal(err)
	}

	if stats.Status != "disconnected" {
		t.Errorf("status = %s", stats.Status)
	}
	if stats.Duration != 0 {
		t.Errorf("duration = %d", stats.Duration)
	}

	t.Logf("OK: stats = %s", statsJson)
}

func TestPing(t *testing.T) {
	cb := &testCallback{}
	engine := NewEngine(cb)

	ping := engine.Ping("8.8.8.8", 53)

	if ping > 0 {
		t.Logf("OK: ping 8.8.8.8:53 = %dms", ping)
	} else {
		t.Logf("OK: ping failed (expected in CI): %d", ping)
	}
}

func TestConnectBadConfig(t *testing.T) {
	cb := &testCallback{}
	engine := NewEngine(cb)

	err := engine.Connect("invalid json")
	if err == nil {
		t.Error("should fail with bad json")
	}

	t.Logf("OK: bad config rejected: %v", err)
}

func TestConnectEmptyServer(t *testing.T) {
	cb := &testCallback{}
	engine := NewEngine(cb)

	cfg := `{"server_addr":"","server_port":8443}`
	err := engine.Connect(cfg)
	if err == nil {
		t.Error("should fail with empty server")
	}

	t.Logf("OK: empty server rejected: %v", err)
}

func TestDisconnectWhenNotConnected(t *testing.T) {
	cb := &testCallback{}
	engine := NewEngine(cb)

	err := engine.Disconnect()
	if err != nil {
		t.Errorf("disconnect when not connected should not error: %v", err)
	}

	t.Log("OK: disconnect when not connected is safe")
}

func TestConnectConfig(t *testing.T) {
	cfg := ConnectConfig{
		ServerAddr:   "1.2.3.4",
		ServerPort:   8443,
		ListenAddr:   "127.0.0.1:1080",
		CoverEnabled: true,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var parsed ConnectConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.ServerAddr != cfg.ServerAddr {
		t.Error("server addr mismatch")
	}
	if parsed.ServerPort != cfg.ServerPort {
		t.Error("port mismatch")
	}

	t.Logf("OK: config json = %s", string(data))
}

func TestCallbackReceived(t *testing.T) {
	cb := &testCallback{}
	engine := NewEngine(cb)

	engine.setStatus("testing")
	engine.log("test message %d", 42)

	if len(cb.statuses) != 1 || cb.statuses[0] != "testing" {
		t.Error("status callback not received")
	}

	if len(cb.logs) != 1 {
		t.Error("log callback not received")
	}

	t.Logf("OK: callbacks received: status=%v logs=%v",
		cb.statuses, cb.logs)
}
