package interleave

import (
	"log"
	"net"
)

// Relay — ریله بین Interleaver و اتصال TCP
// ✅ اصلاح: از Send استفاده می‌شود (با cover traffic)
func Relay(il *Interleaver, conn net.Conn) {
	ch := make(chan error, 2)

	// conn → interleaver (با cover traffic)
	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				// ✅ قبلاً SendDirect بود — بدون cover
				// حالا Send هست — با cover traffic و padding
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

	err := <-ch
	_ = err
	log.Printf("[relay] connection ended")

	conn.Close()
	il.Close()
}
