package interleave

import (
	"net"
)

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

	err1 := <-ch
	conn.Close()
	il.Close()
	err2 := <-ch

	_ = err1
	_ = err2
}
