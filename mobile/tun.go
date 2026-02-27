package mobile

import (
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var tunRunning bool

func (e *Engine) StartTun(fd int32, socksPort int32) error {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in StartTun: %v\n%s", r, debug.Stack()))
		}
	}()

	e.log(fmt.Sprintf("StartTun called: fd=%d socksPort=%d", fd, socksPort))

	// همیشه اول stop کن برای clean state
	e.log("Ensuring clean state...")
	func() {
		defer func() { recover() }()
		if tunRunning {
			engine.Stop()
			tunRunning = false
		}
	}()

	if fd < 0 {
		return fmt.Errorf("invalid fd: %d", fd)
	}

	// صبر کن تا SOCKS5 proxy آماده بشه
	proxy := fmt.Sprintf("127.0.0.1:%d", socksPort)
	e.log(fmt.Sprintf("Waiting for SOCKS5 on %s...", proxy))

	ready := false
	for i := 0; i < 60; i++ { // حداکثر ۳۰ ثانیه
		conn, err := net.DialTimeout("tcp", proxy, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		if i%10 == 9 {
			e.log(fmt.Sprintf("  Still waiting... %d/30s", (i+1)/2))
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		e.log("SOCKS5 not ready after 30s")
		return fmt.Errorf("SOCKS5 proxy not ready on port %d", socksPort)
	}
	e.log("SOCKS5 ready ✅")

	device := fmt.Sprintf("fd://%d", fd)
	proxyURL := fmt.Sprintf("socks5://%s", proxy)
	e.log(fmt.Sprintf("tun2socks: device=%s proxy=%s", device, proxyURL))

	key := &engine.Key{
		Device:   device,
		Proxy:    proxyURL,
		MTU:      1500,
		LogLevel: "warning",
	}

	e.log("engine.Insert()...")
	engine.Insert(key)

	e.log("engine.Start()...")
	engine.Start()

	tunRunning = true
	e.log("TUN started ✅ — all traffic routed")
	return nil
}

func (e *Engine) StopTun() {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in StopTun: %v", r))
		}
	}()

	if !tunRunning {
		return
	}
	e.log("Stopping TUN...")
	engine.Stop()
	tunRunning = false
	e.log("TUN stopped ✅")
}
