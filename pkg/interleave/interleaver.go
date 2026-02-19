package interleave

import (
	"context"
	"log"
	"sync"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

type Interleaver struct {
	sc       *transport.SecureConn
	coverMgr *cover.Manager
	shaper   *cover.Shaper
	sendCh   chan []byte
	mu       sync.Mutex
	seq      uint32
}

func New(sc *transport.SecureConn, coverMgr *cover.Manager) *Interleaver {
	var shaper *cover.Shaper
	if coverMgr != nil {
		shaper = cover.NewShaper(coverMgr.Stats(), cover.PatternWebBrowsing)
	}

	return &Interleaver{
		sc:       sc,
		coverMgr: coverMgr,
		shaper:   shaper,
		sendCh:   make(chan []byte, 64),
	}
}

func (il *Interleaver) Run(ctx context.Context) {
	go il.sendLoop(ctx)
	go il.idleLoop(ctx)
}

func (il *Interleaver) sendLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-il.sendCh:
			il.sendWithCover(data)
		}
	}
}

func (il *Interleaver) idleLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if il.shaper != nil {
				delay := il.shaper.IdleDelay()
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
				}

				if il.shaper.ShouldSendPadding() {
					il.sendPadding()
				}

				if il.coverMgr != nil {
					il.coverMgr.SendOne()
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case <-time.After(3 * time.Second):
				}
			}
		}
	}
}

func (il *Interleaver) sendWithCover(data []byte) {
	if il.coverMgr != nil {
		il.coverMgr.SendOne()
	}

	if il.shaper != nil {
		delay := il.shaper.Delay()
		time.Sleep(delay)
	}

	il.mu.Lock()
	il.seq++
	seq := il.seq
	il.mu.Unlock()

	var pkt *protocol.Packet
	var err error

	if il.shaper != nil {
		padSize := il.shaper.PaddingSize(len(data))
		totalSize := protocol.HeaderSize + len(data) + padSize
		pkt, err = protocol.NewPaddedDataPacket(data, seq, totalSize)
	} else {
		pkt, err = protocol.NewDataPacket(data, seq)
	}

	if err != nil {
		log.Printf("[interleave] packet error: %v", err)
		return
	}

	if err := il.sc.SendPacket(pkt); err != nil {
		log.Printf("[interleave] send error: %v", err)
		return
	}

	if il.shaper != nil {
		delay := il.shaper.Delay()
		time.Sleep(delay / 2)
	}

	if il.coverMgr != nil {
		il.coverMgr.SendOne()
	}
}

func (il *Interleaver) sendPadding() {
	il.mu.Lock()
	il.seq++
	seq := il.seq
	il.mu.Unlock()

	size := 64
	if il.shaper != nil {
		size = il.shaper.FragmentSize()
		if size > 1024 {
			size = 1024
		}
		if size < 16 {
			size = 16
		}
	}

	pkt, err := protocol.NewPaddingPacket(size, seq)
	if err != nil {
		return
	}

	il.sc.SendPacket(pkt)
}

func (il *Interleaver) Send(data []byte) {
	il.sendCh <- data
}

func (il *Interleaver) SendDirect(data []byte) error {
	il.mu.Lock()
	il.seq++
	seq := il.seq
	il.mu.Unlock()

	pkt, err := protocol.NewDataPacket(data, seq)
	if err != nil {
		return err
	}
	return il.sc.SendPacket(pkt)
}

func (il *Interleaver) Recv() ([]byte, error) {
	for {
		pkt, err := il.sc.RecvPacket()
		if err != nil {
			return nil, err
		}

		switch pkt.Type {
		case protocol.PacketTypeData:
			return pkt.Payload, nil
		case protocol.PacketTypePadding:
			continue
		case protocol.PacketTypePing:
			il.mu.Lock()
			il.seq++
			seq := il.seq
			il.mu.Unlock()
			pong := protocol.NewPongPacket(seq)
			il.sc.SendPacket(pong)
			continue
		case protocol.PacketTypePong:
			continue
		case protocol.PacketTypeClose:
			return nil, protocol.ErrConnectionClosed
		default:
			return pkt.Payload, nil
		}
	}
}

func (il *Interleaver) Close() error {
	il.mu.Lock()
	il.seq++
	seq := il.seq
	il.mu.Unlock()

	closePkt := protocol.NewClosePacket(seq)
	il.sc.SendPacket(closePkt)
	return il.sc.Close()
}
