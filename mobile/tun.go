package mobile

import (
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var tunRunning bool

func (e *Engine) StartTun(fd int32, socksPort int32) (retErr error) {
	// recover از هر panic
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC in StartTun: %v\n%s", r, debug.Stack())
			e.log(msg)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	e.log(fmt.Sprintf("StartTun called: fd=%d socksPort=%d", fd, socksPort))

	// اول stop قبلی
	func() {
		defer func() { recover() }()
		if tunRunning {
			e.log("Stopping previous TUN...")
			engine.Stop()
			tunRunning = false
			time.Sleep(500 * time.Millisecond)
		}
	}()

	if fd < 0 {
		return fmt.Errorf("invalid fd: %d", fd)
	}

	// صبر کن SOCKS5 آماده بشه
	proxy := fmt.Sprintf("127.0.0.1:%d", socksPort)
	e.log(fmt.Sprintf("Waiting for SOCKS5 on %s...", proxy))

	ready := false
	for i := 0; i < 120; i++ { // 60 ثانیه
		conn, err := net.DialTimeout("tcp", proxy, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		if i%20 == 19 {
			e.log(fmt.Sprintf("  Still waiting for SOCKS5... %ds", (i+1)/2))
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		e.log("SOCKS5 not ready after 60s — aborting TUN")
		return fmt.Errorf("SOCKS5 not ready on %s", proxy)
	}
	e.log("SOCKS5 ready ✅")

	device := fmt.Sprintf("fd://%d", fd)
	proxyURL := fmt.Sprintf("socks5://%s", proxy)
	e.log(fmt.Sprintf("tun2socks config: device=%s proxy=%s mtu=1500", device, proxyURL))

	key := &engine.Key{
		Device:   device,
		Proxy:    proxyURL,
		MTU:      1500,
		LogLevel: "warning",
	}

	e.log("engine.Insert()...")
	engine.Insert(key)
	e.log("engine.Insert() done ✅")

	// engine.Start() رو توی goroutine بزن — اگه block کنه یا panic بکنه main thread رو نکشه
	e.log("engine.Start() in goroutine...")
	done := make(chan struct{}, 1)
	panicCh := make(chan string, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- fmt.Sprintf("%v\n%s", r, debug.Stack())
			}
		}()
		engine.Start()
		done <- struct{}{}
	}()

	// ۵ ثانیه صبر کن
	select {
	case <-done:
		e.log("engine.Start() completed ✅")
	case msg := <-panicCh:
		e.log(fmt.Sprintf("engine.Start() PANIC: %s", msg))
		return fmt.Errorf("tun2socks panic: %s", msg)
	case <-time.After(5 * time.Second):
		// ← engine.Start() ممکنه non-blocking باشه و فوری return کنه
		// یا blocking باشه — هردو اوکیه
		e.log("engine.Start() still running (background) — OK")
	}

	tunRunning = true
	e.log("TUN started ✅ — traffic routed through SOCKS5")
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
