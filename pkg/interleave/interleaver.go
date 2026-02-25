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
	// ✅ M20: حداقل delay برای idleLoop — جلوگیری از busy-loop
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
	if coverMgr != nil {
		shaper = cover.NewShaper(coverMgr.Stats(), cover.PatternWebBrowsing)
	}

	il := &Interleaver{
		sc:       sc,
		coverMgr: coverMgr,
		shaper:   shaper,
		sendCh:   make(chan []byte, 128),
	}
	// ✅ Start seq after handshake auth to avoid replay detection
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
		case data := <-il.sendCh:
			il.sendShaped(data)
		}
	}
}

func (il *Interleaver) sendShaped(data []byte) {
	if il.shaper != nil {
		delay := il.shaper.Delay()
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	seq := il.seq.Add(1)

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
	}
}

// ✅ M20: idleLoop با حداقل delay تضمین‌شده
// قبلاً: اگه IdleDelay()=0 برمیگردوند → busy-loop + CPU 100%
// الان: حداقل 100ms delay
func (il *Interleaver) idleLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if il.shaper != nil {
			delay := il.shaper.IdleDelay()
			// ✅ M20: حداقل delay
			if delay < minIdleDelay {
				delay = minIdleDelay
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
	}

	pkt, err := protocol.NewPaddingPacket(size, seq)
	if err != nil {
		return
	}

	il.sc.SendPacket(pkt)
}

func (il *Interleaver) Send(data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)

	select {
	case il.sendCh <- cp:
	default:
		// ✅ M18: fallback هم shaped بفرسته (نه direct)
		// قبلاً: SendDirect → بدون padding/timing → fingerprint-able
		// الان: sendShaped → padding + timing حفظ میشه
		log.Printf("[interleave] send channel full, sending shaped directly")
		il.sendShaped(cp)
	}
}

func (il *Interleaver) SendDirect(data []byte) error {
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
			// ✅ M24: Pong با SeqNum پینگ — نه seq جدید
			// قبلاً: seq جدید → receiver نمیتونه match کنه
			// الان: echo back → sender میتونه RTT حساب کنه
			pong := protocol.NewPongPacket(pkt.SeqNum)
			il.sc.SendPacket(pong)
			continue
		case protocol.PacketTypePong:
			// ✅ M24: فعلاً log — بعداً میشه RTT tracking اضافه کرد
			continue
		case protocol.PacketTypeClose:
			return nil, protocol.ErrConnectionClosed
		default:
			return pkt.Payload, nil
		}
	}
}

func (il *Interleaver) Close() error {
	return il.sc.Close()
}
