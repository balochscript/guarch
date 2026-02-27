package mobile

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

var (
	tunRunning bool
	tunCancel  context.CancelFunc
	tunMu      sync.Mutex
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
	goLog(fmt.Sprintf(">>> StartTun fd=%d socksPort=%d", fd, socksPort))
	e.log(fmt.Sprintf("StartTun: fd=%d socksPort=%d", fd, socksPort))

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC: %v\n%s", r, debug.Stack())
			goLog(msg)
			e.log(msg)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	tunMu.Lock()
	defer tunMu.Unlock()

	// قبلی رو ببند
	if tunRunning && tunCancel != nil {
		goLog("Stopping previous TUN...")
		tunCancel()
		tunRunning = false
		time.Sleep(300 * time.Millisecond)
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

	// dup fd — یه کپی بگیر تا اگه Java بست، Go هنوز داشته باشه
	goLog("Duplicating fd...")
	newFd, err := unix.Dup(int(fd))
	if err != nil {
		goLog(fmt.Sprintf("dup failed: %v, using original", err))
		newFd = int(fd)
	} else {
		goLog(fmt.Sprintf("dup: %d → %d", fd, newFd))
	}

	// non-blocking
	unix.SetNonblock(newFd, true)

	tunFile := os.NewFile(uintptr(newFd), "tun")
	if tunFile == nil {
		return fmt.Errorf("NewFile nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	tunCancel = cancel
	tunRunning = true

	// شروع packet relay
	go e.packetRelay(ctx, tunFile, proxy)

	goLog("TUN relay started ✅")
	e.log("TUN started ✅")
	return nil
}

// packetRelay — خوندن IP packet از TUN و فوروارد از طریق SOCKS5
func (e *Engine) packetRelay(ctx context.Context, tunFile *os.File, proxy string) {
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC in packetRelay: %v", r))
		}
		tunFile.Close()
		goLog("packetRelay ended")
	}()

	goLog("packetRelay started")
	buf := make([]byte, 65535)
	connTracker := &connTracker{
		conns: make(map[string]net.Conn),
		proxy: proxy,
	}
	defer connTracker.closeAll()

	for {
		select {
		case <-ctx.Done():
			goLog("packetRelay: context done")
			return
		default:
		}

		// خوندن از TUN (با timeout تا cancel چک بشه)
		tunFile.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := tunFile.Read(buf)
		if err != nil {
			if os.IsTimeout(err) {
				continue
			}
			if err == io.EOF {
				goLog("TUN EOF")
				return
			}
			continue
		}

		if n < 20 {
			continue
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		// IPv4 فقط
		if packet[0]>>4 != 4 {
			continue
		}

		proto := packet[9]
		ihl := int(packet[0]&0x0f) * 4

		switch proto {
		case 6: // TCP
			if len(packet) < ihl+20 {
				continue
			}
			srcPort := int(packet[ihl])<<8 | int(packet[ihl+1])
			dstIP := net.IPv4(packet[16], packet[17], packet[18], packet[19])
			dstPort := int(packet[ihl+2])<<8 | int(packet[ihl+3])

			key := fmt.Sprintf("%d→%s:%d", srcPort, dstIP, dstPort)

			// SYN packet → اتصال جدید
			flags := packet[ihl+13]
			if flags&0x02 != 0 { // SYN
				go connTracker.handleTCP(key, dstIP.String(), dstPort, proxy, tunFile, packet)
			}

		case 17: // UDP
			if len(packet) < ihl+8 {
				continue
			}
			dstIP := net.IPv4(packet[16], packet[17], packet[18], packet[19])
			dstPort := int(packet[ihl+2])<<8 | int(packet[ihl+3])

			// DNS query → forward مستقیم
			if dstPort == 53 {
				go handleDNS(packet[ihl+8:n], dstIP.String(), tunFile, packet)
			}
		}
	}
}

// connTracker — ترک کردن TCP connections
type connTracker struct {
	mu    sync.Mutex
	conns map[string]net.Conn
	proxy string
}

func (ct *connTracker) handleTCP(key string, dstIP string, dstPort int, proxy string, tunFile *os.File, origPacket []byte) {
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC handleTCP: %v", r))
		}
	}()

	target := fmt.Sprintf("%s:%d", dstIP, dstPort)

	// SOCKS5 connect
	socksConn, err := net.DialTimeout("tcp", proxy, 5*time.Second)
	if err != nil {
		goLog(fmt.Sprintf("SOCKS5 dial failed for %s: %v", target, err))
		return
	}

	// SOCKS5 handshake
	socksConn.Write([]byte{0x05, 0x01, 0x00})
	resp := make([]byte, 2)
	io.ReadFull(socksConn, resp)
	if resp[1] != 0x00 {
		socksConn.Close()
		return
	}

	// SOCKS5 connect request
	ip := net.ParseIP(dstIP).To4()
	req := []byte{0x05, 0x01, 0x00, 0x01}
	req = append(req, ip...)
	req = append(req, byte(dstPort>>8), byte(dstPort))
	socksConn.Write(req)

	connResp := make([]byte, 10)
	io.ReadFull(socksConn, connResp)
	if connResp[1] != 0x00 {
		socksConn.Close()
		return
	}

	ct.mu.Lock()
	ct.conns[key] = socksConn
	ct.mu.Unlock()

	goLog(fmt.Sprintf("TCP connected: %s ✅", target))

	// relay
	go func() {
		defer socksConn.Close()
		buf := make([]byte, 32768)
		for {
			n, err := socksConn.Read(buf)
			if n > 0 {
				// TODO: properly construct IP response packet and write to tunFile
				// For now, just log
			}
			if err != nil {
				break
			}
		}
		ct.mu.Lock()
		delete(ct.conns, key)
		ct.mu.Unlock()
	}()
}

func (ct *connTracker) closeAll() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	for k, c := range ct.conns {
		c.Close()
		delete(ct.conns, k)
	}
}

// handleDNS — DNS query forward
func handleDNS(payload []byte, dstIP string, tunFile *os.File, origPacket []byte) {
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC handleDNS: %v", r))
		}
	}()

	if len(payload) < 12 {
		return
	}

	// Forward DNS to 8.8.8.8
	conn, err := net.DialTimeout("udp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	conn.Write(payload)

	resp := make([]byte, 65535)
	n, err := conn.Read(resp)
	if err != nil || n < 12 {
		return
	}

	// TODO: construct proper IP/UDP response packet and write to tunFile
	// For now DNS response goes directly
}

func (e *Engine) StopTun() {
	initGoLog()
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC StopTun: %v", r))
		}
	}()

	tunMu.Lock()
	defer tunMu.Unlock()

	if !tunRunning {
		return
	}
	goLog("StopTun")
	e.log("Stopping TUN...")
	if tunCancel != nil {
		tunCancel()
	}
	tunRunning = false
	goLog("TUN stopped ✅")
	e.log("TUN stopped ✅")
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
