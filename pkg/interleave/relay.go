package interleave

import (
	"log"
	"net"
)

func Relay(il *Interleaver, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if serr := il.SendDirect(buf[:n]); serr != nil {
					ch <- serr
					return
				}
			}
			if err != nil {
				ch <- err
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
