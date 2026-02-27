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

	e.log(fmt.Sprintf("StartTun: fd=%d socksPort=%d", fd, socksPort))

	if fd < 0 {
		err := fmt.Errorf("invalid fd: %d", fd)
		e.log(err.Error())
		return err
	}

	// چک fd معتبره
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		e.log(fmt.Sprintf("fd %d invalid: %v", fd, err))
		return fmt.Errorf("fd not valid: %v", err)
	}

	f := os.NewFile(uintptr(fd), "tun")
	if f == nil {
		err := fmt.Errorf("os.NewFile nil for fd=%d", fd)
		e.log(err.Error())
		return err
	}
	e.log(fmt.Sprintf("fd=%d valid ✅", fd))

	proxy := fmt.Sprintf("socks5://127.0.0.1:%d", socksPort)
	device := fmt.Sprintf("fd://%d", fd)
	e.log(fmt.Sprintf("tun2socks: device=%s proxy=%s", device, proxy))

	key := &engine.Key{
		Device:   device,
		Proxy:    proxy,
		MTU:      1500,
		LogLevel: "debug",
	}

	e.log("engine.Insert()...")
	engine.Insert(key)
	e.log("engine.Insert() done ✅")

	// engine.Start() مقدار برنمیگردونه — فقط صداش بزن
	e.log("engine.Start()...")
	startDone := make(chan struct{}, 1)
	startPanic := make(chan string, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				startPanic <- fmt.Sprintf("PANIC: %v\n%s", r, debug.Stack())
			}
		}()
		engine.Start()
		startDone <- struct{}{}
	}()

	// یکم صبر کن ببین panic میکنه یا نه
	select {
	case msg := <-startPanic:
		e.log("engine.Start() " + msg)
		return fmt.Errorf("engine.Start panic")
	case <-startDone:
		e.log("engine.Start() done ✅")
	}

	tunRunning = true
	e.log("TUN started ✅")
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
