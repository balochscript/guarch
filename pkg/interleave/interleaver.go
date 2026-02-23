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

	return &Interleaver{
		sc:       sc,
		coverMgr: coverMgr,
		shaper:   shaper,
		sendCh:   make(chan []byte, 128),
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
			il.sendShaped(data)
		}
	}
}

// ✅ FIX B4: حذف Cover از send path
// قبلاً: sendWithCover() → ۲ بار Cover + blocking HTTP!
// الان: فقط padding + timing → سریع‌تر
func (il *Interleaver) sendShaped(data []byte) {
	// فقط timing jitter (نه Cover!)
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
		return
	}
}

// ✅ FIX B4: idleLoop فقط padding می‌فرسته، Cover نه
// Cover Manager مستقل و جداگانه اجرا می‌شه
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
				// ✅ حذف: il.coverMgr.SendOne()
				// Cover Manager خودش مستقل کار می‌کنه
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

// ✅ FIX A2: Send() حالا copy می‌کنه!
func (il *Interleaver) Send(data []byte) {
	// کپی داده قبل از فرستادن به channel
	// جلوگیری از race condition وقتی caller بافر رو reuse می‌کنه
	cp := make([]byte, len(data))
	copy(cp, data)

	select {
	case il.sendCh <- cp:
	default:
		// بافر پره — مستقیم بفرست (بدون shaping)
		log.Printf("[interleave] send channel full, sending direct")
		il.SendDirect(cp)
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
			seq := il.seq.Add(1)
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
	return il.sc.Close()
}
