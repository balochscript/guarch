package mobile

import (
	"fmt"
	"os"
	"runtime/debug"
	"syscall"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var tunRunning bool

func (e *Engine) StartTun(fd int32, socksPort int32) error {
	// ← recover از Go panic
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC in StartTun: %v\n%s", r, debug.Stack())
			e.log(msg)
		}
	}()

	if tunRunning {
		e.log("TUN already running, stopping first...")
		e.StopTun()
	}

	e.log(fmt.Sprintf("StartTun called: fd=%d socksPort=%d", fd, socksPort))

	// ← چک کن fd معتبره
	e.log("Checking fd validity...")
	if fd < 0 {
		err := fmt.Errorf("invalid fd: %d", fd)
		e.log(err.Error())
		return err
	}

	// ← چک کن fd واقعاً باز هست
	e.log("Checking fd is open...")
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		e.log(fmt.Sprintf("fd %d not valid (SetNonblock failed): %v", fd, err))
		return fmt.Errorf("fd not valid: %v", err)
	}

	// ← چک فایل
	f := os.NewFile(uintptr(fd), "tun")
	if f == nil {
		err := fmt.Errorf("os.NewFile returned nil for fd=%d", fd)
		e.log(err.Error())
		return err
	}
	// نکته: NewFile رو نمیبندیم چون tun2socks ازش استفاده میکنه
	e.log(fmt.Sprintf("fd=%d is valid ✅", fd))

	proxy := fmt.Sprintf("socks5://127.0.0.1:%d", socksPort)
	device := fmt.Sprintf("fd://%d", fd)

	e.log(fmt.Sprintf("Configuring tun2socks: device=%s proxy=%s", device, proxy))

	key := &engine.Key{
		Device:   device,
		Proxy:    proxy,
		MTU:      1500,
		LogLevel: "debug", // ← debug برای لاگ بیشتر
	}

	e.log("Calling engine.Insert()...")
	engine.Insert(key)
	e.log("engine.Insert() done ✅")

	e.log("Calling engine.Start()...")

	// ← engine.Start() رو توی goroutine با recover اجرا کن
	startErr := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("PANIC in engine.Start(): %v\n%s", r, debug.Stack())
				e.log(msg)
				startErr <- fmt.Errorf("engine.Start panic: %v", r)
			}
		}()

		err := engine.Start()
		if err != nil {
			e.log(fmt.Sprintf("engine.Start() returned error: %v", err))
			startErr <- err
		} else {
			e.log("engine.Start() returned OK ✅")
			startErr <- nil
		}
	}()

	// ← منتظر بمون ببین Start موفق بود یا نه
	e.log("Waiting for engine.Start() result...")
	select {
	case err := <-startErr:
		if err != nil {
			e.log(fmt.Sprintf("TUN start FAILED: %v", err))
			return err
		}
	}

	tunRunning = true
	e.log("TUN handler started ✅ — all traffic routed")
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
	e.log("Stopping TUN handler...")
	engine.Stop()
	tunRunning = false
	e.log("TUN handler stopped ✅")
}
