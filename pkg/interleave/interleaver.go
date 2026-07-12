package interleave

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

const (
	minIdleDelay = 100 * time.Millisecond
)

type Interleaver struct {
	sc       *transport.SecureConn
	coverMgr *cover.Manager
	shaper   *cover.Shaper
	sendCh   chan []byte
	seq      atomic.Uint32
}

func New(sc *transport.SecureConn, coverMgr *cover.Manager) *Interleaver {
	var shaper *cover.Shaper
	if coverMgr != nil && coverMgr.IsRunning() {
		shaper = cover.NewShaper(coverMgr.Stats(), cover.PatternWebBrowsing)
	}
	il := &Interleaver{
		sc:       sc,
		coverMgr: coverMgr,
		shaper:   shaper,
		sendCh:   make(chan []byte, 256),
	}
	il.seq.Store(sc.SendSeqNum())
	return il
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
		case data, ok := <-il.sendCh:
			if !ok {
				return
			}
			il.sendShaped(data)
		}
	}
}

func (il *Interleaver) sendShaped(data []byte) {
	if il.shaper != nil {
		delay := il.shaper.Delay()
		if delay > 0 && delay < 5*time.Second {
			time.Sleep(delay)
		}
	}
	seq := il.seq.Add(1)
	var pkt *protocol.Packet
	var err error
	if il.shaper != nil {
		padSize := il.shaper.PaddingSize(len(data))
		if padSize < 0 {
			padSize = 0
		}
		if padSize > 1024 {
			padSize = 1024
		}
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
	}
}

func (il *Interleaver) idleLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if il.shaper != nil {
			delay := il.shaper.IdleDelay()
			if delay < minIdleDelay {
				delay = minIdleDelay
			}
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			if il.shaper.ShouldSendPadding() {
				il.sendPadding()
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

func (il *Interleaver) sendPadding() {
	seq := il.seq.Add(1)
	size := 64
	if il.shaper != nil {
		size = il.shaper.FragmentSize()
		if size < 64 {
			size = 64
		}
		if size > 1024 {
			size = 1024
		}
	}
	pkt, err := protocol.NewPaddingPacket(size, seq)
	if err != nil {
		return
	}
	_ = il.sc.SendPacket(pkt)
}

func (il *Interleaver) Send(data []byte) {
	if len(data) == 0 {
		return
	}
	if len(data) > 1024*1024 {
		return
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	select {
	case il.sendCh <- cp:
	default:
		log.Printf("[interleave] send channel full (%d), sending shaped directly", len(il.sendCh))
		il.sendShaped(cp)
	}
}

func (il *Interleaver) SendDirect(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	seq := il.seq.Add(1)
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
			pong := protocol.NewPongPacket(pkt.SeqNum)
			_ = il.sc.SendPacket(pong)
			continue
		case protocol.PacketTypePong:
			continue
		case protocol.PacketTypeClose:
			return nil, protocol.ErrConnectionClosed
		default:
			if len(pkt.Payload) > 0 {
				return pkt.Payload, nil
			}
			continue
		}
	}
}

func (il *Interleaver) Close() error {
	close(il.sendCh)
	return il.sc.Close()
}
