package mobile

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var tunRunning bool

// logFile برای لاگ مستقیم به فایل (حتی اگه process کرش کنه)
var goLogFile *os.File

func initGoLog() {
	if goLogFile != nil {
		return
	}
	// لاگ توی /data/data/com.guarch.app/files/go_debug.log
	// از Android file path استفاده میکنیم
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
		goLogFile.Sync() // فوری flush
	}
}

func (e *Engine) StartTun(fd int32, socksPort int32) (retErr error) {
	initGoLog()
	goLog(fmt.Sprintf(">>> StartTun fd=%d socksPort=%d", fd, socksPort))
	e.log(fmt.Sprintf("StartTun: fd=%d socksPort=%d", fd, socksPort))

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC: %v\n%s", r, debug.Stack())
			goLog("StartTun " + msg)
			e.log("StartTun " + msg)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	// Stop قبلی
	func() {
		defer func() { recover() }()
		if tunRunning {
			goLog("Stopping previous TUN...")
			engine.Stop()
			tunRunning = false
			time.Sleep(500 * time.Millisecond)
			goLog("Previous TUN stopped")
		}
	}()

	if fd < 0 {
		goLog(fmt.Sprintf("ERROR: invalid fd %d", fd))
		return fmt.Errorf("invalid fd: %d", fd)
	}

	// ═══ Step 1: چک fd ═══
	goLog("Step 1: Checking fd...")
	f := os.NewFile(uintptr(fd), "tun-test")
	if f == nil {
		goLog("ERROR: os.NewFile returned nil")
		return fmt.Errorf("os.NewFile nil")
	}
	// فقط چک — نبند! tun2socks خودش استفاده میکنه
	stat, err := f.Stat()
	if err != nil {
		goLog(fmt.Sprintf("WARNING: fd stat error: %v (may be normal for tun)", err))
	} else {
		goLog(fmt.Sprintf("fd stat: name=%s size=%d", stat.Name(), stat.Size()))
	}
	// بذار f بره — GC نباید close کنه چون detachFd بوده
	f = nil
	goLog("Step 1: fd OK ✅")

	// ═══ Step 2: صبر برای SOCKS5 ═══
	proxy := fmt.Sprintf("127.0.0.1:%d", socksPort)
	goLog(fmt.Sprintf("Step 2: Waiting for SOCKS5 on %s...", proxy))

	ready := false
	for i := 0; i < 120; i++ {
		conn, err := net.DialTimeout("tcp", proxy, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		if i%20 == 0 && i > 0 {
			goLog(fmt.Sprintf("  Still waiting... %ds (err: %v)", i/2, err))
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		goLog("ERROR: SOCKS5 not ready after 60s")
		e.log("SOCKS5 not ready — TUN aborted")
		return fmt.Errorf("SOCKS5 not ready on %s", proxy)
	}
	goLog("Step 2: SOCKS5 ready ✅")

	// ═══ Step 3: Test SOCKS5 ═══
	goLog("Step 3: Testing SOCKS5 connection...")
	testConn, err := net.DialTimeout("tcp", proxy, 2*time.Second)
	if err != nil {
		goLog(fmt.Sprintf("SOCKS5 test failed: %v", err))
	} else {
		goLog("SOCKS5 test connection OK ✅")
		testConn.Close()
	}

	// ═══ Step 4: tun2socks config ═══
	device := fmt.Sprintf("fd://%d", fd)
	proxyURL := fmt.Sprintf("socks5://%s", proxy)
	goLog(fmt.Sprintf("Step 4: device=%s proxy=%s mtu=1500", device, proxyURL))

	key := &engine.Key{
		Device:   device,
		Proxy:    proxyURL,
		MTU:      1500,
		LogLevel: "silent",
	}

	goLog("Step 4a: engine.Insert()...")
	engine.Insert(key)
	goLog("Step 4a: engine.Insert() done ✅")

	// ═══ Step 5: engine.Start() — خطرناک! ═══
	goLog("Step 5: engine.Start() — THIS IS WHERE CRASH MIGHT HAPPEN")
	goLog("Step 5: Starting in goroutine with timeout...")

	panicCh := make(chan string, 1)
	doneCh := make(chan struct{}, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("PANIC in engine.Start(): %v\n%s", r, debug.Stack())
				goLog(msg)
				panicCh <- msg
			}
		}()
		goLog("Step 5: calling engine.Start()...")
		engine.Start()
		goLog("Step 5: engine.Start() returned ✅")
		doneCh <- struct{}{}
	}()

	select {
	case <-doneCh:
		goLog("Step 5: engine.Start() completed ✅")
	case msg := <-panicCh:
		goLog("Step 5: engine.Start() PANICKED: " + msg)
		e.log("TUN PANIC: " + msg)
		return fmt.Errorf("tun2socks panic")
	case <-time.After(10 * time.Second):
		goLog("Step 5: engine.Start() timeout (10s) — assuming running in background")
	}

	tunRunning = true
	goLog("TUN STARTED ✅ — all steps completed")
	e.log("TUN started ✅")
	return nil
}

func (e *Engine) StopTun() {
	initGoLog()
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC in StopTun: %v", r))
		}
	}()

	if !tunRunning {
		return
	}
	goLog("StopTun called")
	e.log("Stopping TUN...")
	engine.Stop()
	tunRunning = false
	goLog("TUN stopped ✅")
	e.log("TUN stopped ✅")
}

// ReadGoLog خوندن لاگ Go از Kotlin
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
	return "No Go log file found"
}

// ClearGoLog پاک کردن لاگ
func ClearGoLog() {
	paths := []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
	}
	for _, p := range paths {
		os.WriteFile(p, []byte(""), 0644)
	}
}

// ═══ Helper: Test read/write on fd ═══
func testFd(fd int32) error {
	f := os.NewFile(uintptr(fd), "tun-rw-test")
	if f == nil {
		return fmt.Errorf("NewFile nil")
	}
	// تست read — TUN باید readable باشه
	buf := make([]byte, 1)
	f.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := f.Read(buf)
	if err != nil && err != io.EOF {
		// timeout اوکیه — یعنی fd بازه ولی دیتایی نیست
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil // اوکیه
		}
		return fmt.Errorf("read test: %v", err)
	}
	return nil
}
