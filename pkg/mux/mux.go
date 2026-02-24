package mux

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

const (
	cmdOpen  byte = 0x01
	cmdClose byte = 0x02
	cmdData  byte = 0x03
	cmdPing  byte = 0x04
	cmdPong  byte = 0x05

	muxHeaderSize = 5
)

type Mux struct {
	sc        *transport.SecureConn
	streams   sync.Map
	nextID    atomic.Uint32
	acceptCh  chan *Stream
	closeCh   chan struct{}
	closeOnce sync.Once
	sendMu    sync.Mutex
	isServer  bool
}

func NewMux(sc *transport.SecureConn, isServer bool) *Mux {
	m := &Mux{
		sc:       sc,
		acceptCh: make(chan *Stream, 32),
		closeCh:  make(chan struct{}),
		isServer: isServer,
	}

	if isServer {
		m.nextID.Store(1_000_000_000)
	}

	go m.readLoop()
	go m.keepAlive()
	return m
}

func (m *Mux) readLoop() {
	defer m.Close()

	for {
		pkt, err := m.sc.RecvPacket()
		if err != nil {
			log.Printf("[mux] read loop ended: %v", err)
			return
		}

		switch pkt.Type {
		case protocol.PacketTypePadding:
			continue
		case protocol.PacketTypePing:
			pong := protocol.NewPongPacket(pkt.SeqNum)
			m.sc.SendPacket(pong)
			continue
		case protocol.PacketTypePong:
			continue
		case protocol.PacketTypeClose:
			log.Printf("[mux] received CLOSE packet")
			return
		case protocol.PacketTypeData:
			m.handleMuxFrame(pkt.Payload)
		default:
			continue
		}
	}
}

func (m *Mux) handleMuxFrame(data []byte) {
	if len(data) < muxHeaderSize {
		return
	}

	cmd := data[0]
	streamID := binary.BigEndian.Uint32(data[1:5])
	payload := data[muxHeaderSize:]

	switch cmd {
	case cmdOpen:
		s := newStream(streamID, m)
		m.streams.Store(streamID, s)
		log.Printf("[mux] accepted stream %d", streamID)
		select {
		case m.acceptCh <- s:
		case <-m.closeCh:
			return
		}

	case cmdData:
		if val, ok := m.streams.Load(streamID); ok {
			s := val.(*Stream)
			if !s.closed.Load() {
				p := make([]byte, len(payload))
				copy(p, payload)
				select {
				case s.readCh <- p:
				case <-s.doneCh:
				case <-m.closeCh:
					return
				}
			}
		}

	case cmdClose:
		if val, ok := m.streams.Load(streamID); ok {
			s := val.(*Stream)
			s.markClosed()
			m.streams.Delete(streamID)
			log.Printf("[mux] stream %d closed by remote", streamID)
		}

	case cmdPing:
		m.sendFrame(cmdPong, streamID, nil)

	case cmdPong:
	}
}

// ✅ L13: time.NewTimer بجای time.After — جلوگیری از timer leak
// قبلاً: time.After → هر iteration یه timer ساخته میشد که تا expire نشه GC نمیشد
// الان: NewTimer + Stop → اگه closeCh زودتر بسته بشه، timer آزاد میشه
func (m *Mux) keepAlive() {
	for {
		jitter := time.Duration(randomMuxInt(25000, 35000)) * time.Millisecond
		timer := time.NewTimer(jitter)
		select {
		case <-m.closeCh:
			timer.Stop()
			return
		case <-timer.C:
			if err := m.sendFrame(cmdPing, 0, nil); err != nil {
				return
			}
		}
	}
}

func (m *Mux) sendFrame(cmd byte, streamID uint32, payload []byte) error {
	m.sendMu.Lock()
	defer m.sendMu.Unlock()

	frame := make([]byte, muxHeaderSize+len(payload))
	frame[0] = cmd
	binary.BigEndian.PutUint32(frame[1:5], streamID)
	if len(payload) > 0 {
		copy(frame[muxHeaderSize:], payload)
	}

	return m.sc.Send(frame)
}

func (m *Mux) OpenStream() (*Stream, error) {
	select {
	case <-m.closeCh:
		return nil, fmt.Errorf("mux: closed")
	default:
	}

	id := m.nextID.Add(1)
	s := newStream(id, m)
	m.streams.Store(id, s)

	if err := m.sendFrame(cmdOpen, id, nil); err != nil {
		m.streams.Delete(id)
		return nil, fmt.Errorf("mux: open: %w", err)
	}

	log.Printf("[mux] opened stream %d", id)
	return s, nil
}

func (m *Mux) AcceptStream() (*Stream, error) {
	select {
	case s, ok := <-m.acceptCh:
		if !ok {
			return nil, fmt.Errorf("mux: closed")
		}
		return s, nil
	case <-m.closeCh:
		return nil, fmt.Errorf("mux: closed")
	}
}

func (m *Mux) Close() error {
	m.closeOnce.Do(func() {
		close(m.closeCh)
		m.streams.Range(func(key, val any) bool {
			s := val.(*Stream)
			s.markClosed()
			m.streams.Delete(key)
			return true
		})
		m.sc.Close()
	})
	return nil
}

func (m *Mux) IsClosed() bool {
	select {
	case <-m.closeCh:
		return true
	default:
		return false
	}
}

// ═══════════════════════════════════════
// Stream
// ═══════════════════════════════════════

type Stream struct {
	id     uint32
	mux    *Mux
	readCh chan []byte
	doneCh chan struct{}
	closed atomic.Bool

	readMu   sync.Mutex
	readBuf  []byte
	doneOnce sync.Once
}

func newStream(id uint32, m *Mux) *Stream {
	return &Stream{
		id:     id,
		mux:    m,
		readCh: make(chan []byte, 256),
		doneCh: make(chan struct{}),
	}
}

func (s *Stream) Read(p []byte) (int, error) {
	s.readMu.Lock()
	if len(s.readBuf) > 0 {
		n := copy(p, s.readBuf)
		s.readBuf = s.readBuf[n:]
		s.readMu.Unlock()
		return n, nil
	}
	s.readMu.Unlock()

	select {
	case data, ok := <-s.readCh:
		if !ok {
			return 0, io.EOF
		}
		n := copy(p, data)
		if n < len(data) {
			s.readMu.Lock()
			s.readBuf = make([]byte, len(data)-n)
			copy(s.readBuf, data[n:])
			s.readMu.Unlock()
		}
		return n, nil
	case <-s.doneCh:
		return 0, io.EOF
	}
}

func (s *Stream) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}

	const maxChunk = 32768
	sent := 0

	for sent < len(p) {
		end := sent + maxChunk
		if end > len(p) {
			end = len(p)
		}
		if err := s.mux.sendFrame(cmdData, s.id, p[sent:end]); err != nil {
			return sent, err
		}
		sent = end
	}

	return sent, nil
}

func (s *Stream) Close() error {
	if s.closed.Swap(true) {
		return nil
	}
	s.mux.sendFrame(cmdClose, s.id, nil)
	s.mux.streams.Delete(s.id)
	s.markClosed()
	return nil
}

func (s *Stream) markClosed() {
	s.closed.Store(true)
	s.doneOnce.Do(func() {
		close(s.doneCh)
	})
}

func (s *Stream) ID() uint32 {
	return s.id
}

// ═══════════════════════════════════════
// RelayStream
// ═══════════════════════════════════════

func RelayStream(stream *Stream, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if _, werr := stream.Write(buf[:n]); werr != nil {
					ch <- werr
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
		buf := make([]byte, 32768)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				if _, werr := conn.Write(buf[:n]); werr != nil {
					ch <- werr
					return
				}
			}
			if err != nil {
				ch <- err
				return
			}
		}
	}()

	<-ch
	stream.Close()
	conn.Close()
	<-ch
}

// ═══════════════════════════════════════
// Helper
// ═══════════════════════════════════════

func randomMuxInt(min, max int) int {
	if max <= min {
		return min
	}
	diff := max - min
	n, err := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	if err != nil {
		return min + diff/2
	}
	return min + int(n.Int64())
}
