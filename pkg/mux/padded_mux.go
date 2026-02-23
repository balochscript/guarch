package mux

import (
	"context"
	"log"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

// PaddedMux — Mux با قابلیت Padding و Timing
// جایگزین NewMux وقتی mode stealth یا balanced باشه
type PaddedMux struct {
	*Mux
	shaper  *cover.Shaper
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewPaddedMux(sc *transport.SecureConn, shaper *cover.Shaper) *PaddedMux {
	ctx, cancel := context.WithCancel(context.Background())
	
	pm := &PaddedMux{
		Mux:    NewMux(sc),
		shaper: shaper,
		ctx:    ctx,
		cancel: cancel,
	}

	if shaper != nil {
		go pm.paddingLoop()
	}

	return pm
}

// paddingLoop — ارسال دوره‌ای padding packets
// فایروال نمی‌تونه بفهمه کِی کاربر فعاله و کِی بی‌کار
func (pm *PaddedMux) paddingLoop() {
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-pm.closeCh:
			return
		default:
			delay := pm.shaper.IdleDelay()
			select {
			case <-pm.ctx.Done():
				return
			case <-pm.closeCh:
				return
			case <-time.After(delay):
			}

			if pm.shaper.ShouldSendPadding() {
				pm.sendPaddingPacket()
			}
		}
	}
}

func (pm *PaddedMux) sendPaddingPacket() {
	size := pm.shaper.FragmentSize()
	pkt, err := protocol.NewPaddingPacket(size, 0)
	if err != nil {
		return
	}
	if err := pm.sc.SendPacket(pkt); err != nil {
		log.Printf("[padded-mux] padding send error: %v", err)
	}
}

func (pm *PaddedMux) Close() error {
	pm.cancel()
	return pm.Mux.Close()
}
