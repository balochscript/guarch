package interleave

import (
	"io"
	"net"
	"time"
)

func Relay(il *Interleaver, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
			n, err := conn.Read(buf)
			if n > 0 {
				il.Send(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					ch <- err
				} else {
					ch <- io.EOF
				}
				return
			}
		}
	}()

	go func() {
		for {
			data, err := il.Recv()
			if err != nil {
				ch <- err
				return
			}
			if len(data) == 0 {
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if _, werr := conn.Write(data); werr != nil {
				ch <- werr
				return
			}
		}
	}()

	<-ch
	_ = conn.Close()
	_ = il.Close()
	go func() {
		<-ch
	}()
}
