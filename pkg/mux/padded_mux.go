package mux

import (
	"context"
	"log"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

type PaddedMux struct {
	*Mux
	shaper *cover.Shaper
	ctx    context.Context
	cancel context.CancelFunc
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

// ✅ C12: paddingLoop اصلاح شده
// قبلاً: default → اولین iteration بدون تأخیر + time.After leak
// الان: time.NewTimer + cleanup + همیشه تأخیر
func (pm *PaddedMux) paddingLoop() {
	for {
		delay := pm.shaper.IdleDelay()
		timer := time.NewTimer(delay)

		select {
		case <-pm.ctx.Done():
			timer.Stop()
			return
		case <-pm.closeCh:
			timer.Stop()
			return
		case <-timer.C:
		}

		if pm.shaper.ShouldSendPadding() {
			pm.sendPaddingPacket()
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
