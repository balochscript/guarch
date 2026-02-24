package interleave

import (
	"log"
	"net"
)

// ✅ M19: Relay حالا هر دو goroutine رو wait میکنه
// قبلاً: فقط اولین خطا خونده میشد → goroutine دوم leak/race
// الان: هر دو خطا drain میشن → هر دو goroutine تمام میشن
func Relay(il *Interleaver, conn net.Conn) {
	ch := make(chan error, 2)

	// conn → interleaver
	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				il.Send(buf[:n])
			}
			if err != nil {
				ch <- err
				return
			}
		}
	}()

	// interleaver → conn
	go func() {
		for {
			data, err := il.Recv()
			if err != nil {
				ch <- err
				return
			}
			if _, werr := conn.Write(data); werr != nil {
				ch <- werr
				return
			}
		}
	}()

	// ✅ M19: اولین خطا → close both → صبر برای دومی
	err1 := <-ch
	conn.Close()
	il.Close()
	err2 := <-ch // صبر تا goroutine دوم هم تمام بشه

	_ = err1
	_ = err2
	// خطاها معمولاً io.EOF/closed هستن — فقط در debug لاگ بزن:
	// log.Printf("[relay] finished: err1=%v err2=%v", err1, err2)
}
