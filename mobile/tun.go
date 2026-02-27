package mobile

import (
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var (
	tunRunning bool
	goLogFile  *os.File
)

func initGoLog() {
	if goLogFile != nil {
		return
	}
	paths := []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
	}
	for _, p := range paths {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			goLogFile = f
			goLog("=== Go logger started ===")
			return
		}
	}
}

func goLog(msg string) {
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05.000"), msg)
	if goLogFile != nil {
		goLogFile.WriteString(line)
		goLogFile.Sync()
	}
}

func (e *Engine) StartTun(fd int32, socksPort int32) (retErr error) {
	initGoLog()
	goLog(fmt.Sprintf(">>> StartTun fd=%d socksPort=%d tunRunning=%v", fd, socksPort, tunRunning))
	e.log(fmt.Sprintf("StartTun: fd=%d socksPort=%d", fd, socksPort))

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC: %v\n%s", r, debug.Stack())
			goLog(msg)
			e.log(msg)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	// اگه قبلاً اجرا شده، دیگه دوباره Start نکن
	if tunRunning {
		goLog("TUN already running — skip")
		e.log("TUN already active")
		return nil
	}

	if fd < 0 {
		return fmt.Errorf("invalid fd: %d", fd)
	}

	// صبر برای SOCKS5
	proxy := fmt.Sprintf("127.0.0.1:%d", socksPort)
	goLog(fmt.Sprintf("Waiting for SOCKS5 on %s...", proxy))

	ready := false
	for i := 0; i < 120; i++ {
		conn, err := net.DialTimeout("tcp", proxy, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		if i%20 == 0 && i > 0 {
			goLog(fmt.Sprintf("  Waiting... %ds", i/2))
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		goLog("SOCKS5 not ready")
		e.log("SOCKS5 not ready")
		return fmt.Errorf("SOCKS5 not ready")
	}
	goLog("SOCKS5 ready ✅")

	device := fmt.Sprintf("fd://%d", fd)
	proxyURL := fmt.Sprintf("socks5://%s", proxy)
	goLog(fmt.Sprintf("tun2socks: device=%s proxy=%s", device, proxyURL))

	key := &engine.Key{
		Device:   device,
		Proxy:    proxyURL,
		MTU:      1500,
		LogLevel: "warning",
	}

	goLog("engine.Insert()...")
	engine.Insert(key)
	goLog("engine.Insert() done")

	goLog("engine.Start()...")
	doneCh := make(chan struct{}, 1)
	panicCh := make(chan string, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- fmt.Sprintf("%v\n%s", r, debug.Stack())
			}
		}()
		engine.Start()
		doneCh <- struct{}{}
	}()

	select {
	case <-doneCh:
		goLog("engine.Start() done ✅")
	case msg := <-panicCh:
		goLog("engine.Start() PANIC: " + msg)
		return fmt.Errorf("tun2socks panic")
	case <-time.After(10 * time.Second):
		goLog("engine.Start() running (background)")
	}

	tunRunning = true
	goLog("TUN STARTED ✅")
	e.log("TUN started ✅")
	return nil
}

// StopTun — هیچوقت engine.Stop() صدا نمیزنیم
func (e *Engine) StopTun() {
	initGoLog()
	goLog("StopTun called — keeping tun2socks alive")
	e.log("TUN kept alive for reconnect")
	// engine.Stop() صدا نمیزنیم!
	// وقتی VPN service بمیره fd بسته میشه و tun2socks خودش error میده
}

func ReadGoLog() string {
	paths := []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return "No Go log"
}

func ClearGoLog() {
	paths := []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
	}
	for _, p := range paths {
		os.WriteFile(p, []byte(""), 0644)
	}
}
