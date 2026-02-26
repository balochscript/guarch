package mobile

import (
	"fmt"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var tunRunning bool

func (e *Engine) StartTun(fd int32, socksPort int32) error {
	if tunRunning {
		e.log("TUN already running")
		return nil
	}

	e.log(fmt.Sprintf("Starting TUN handler (fd=%d → socks5://127.0.0.1:%d)...", fd, socksPort))

	key := &engine.Key{
		Device:   fmt.Sprintf("fd://%d", fd),
		Proxy:    fmt.Sprintf("socks5://127.0.0.1:%d", socksPort),
		MTU:      1500,
		LogLevel: "warning",
	}

	engine.Insert(key)
	engine.Start()
	tunRunning = true

	e.log("TUN handler started ✅ — all traffic routed through tunnel")
	return nil
}

// StopTun stops the TUN handler
func (e *Engine) StopTun() {
	if !tunRunning {
		return
	}
	e.log("Stopping TUN handler...")
	engine.Stop()
	tunRunning = false
	e.log("TUN handler stopped")
}
