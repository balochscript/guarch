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
	
	DefaultMaxStreams     = 1000
	DefaultStreamBuffer   = 256
	DefaultAcceptTimeout  = 5 * time.Minute
	DefaultOpenTimeout    = 10 * time.Second
	DefaultReadTimeout    = 60 * time.Second
	DefaultWriteTimeout   = 30 * time.Second
	DefaultMaxChunkSize   = 32768
)

type MuxConfig struct {
	MaxStreams    int
	StreamBuffer  int
	AcceptTimeout time.Duration
	OpenTimeout   time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
}

func DefaultMuxConfig() *MuxConfig {
	return &MuxConfig{
		MaxStreams:    DefaultMaxStreams,
		StreamBuffer:  DefaultStreamBuffer,
		AcceptTimeout: DefaultAcceptTimeout,
		OpenTimeout:   DefaultOpenTimeout,
		ReadTimeout:   DefaultReadTimeout,
		WriteTimeout:  DefaultWriteTimeout,
	}
}

type Mux struct {
	sc        *transport.SecureConn
	streams   sync.Map
	nextID    atomic.Uint32
	acceptCh  chan *Stream
	closeCh   chan struct{}
	closeOnce sync.Once
	sendMu    sync.Mutex
	isServer  bool
	
	config       *MuxConfig
	streamCount  atomic.Int32
	totalStreams atomic.Uint64
	bytesSent    atomic.Uint64
	bytesRecv    atomic.Uint64
}

func NewMux(sc *transport.SecureConn, isServer bool) *Mux {
	return NewMuxWithConfig(sc, isServer, DefaultMuxConfig())
}

func NewMuxWithConfig(sc *transport.SecureConn, isServer bool, cfg *MuxConfig) *Mux {
	if cfg == nil {
		cfg = DefaultMuxConfig()
	}
	
	m := &Mux{
		sc:       sc,
		acceptCh: make(chan *Stream, 32),
		closeCh:  make(chan struct{}),
		isServer: isServer,
		config:   cfg,
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
		if m.streamCount.Load() >= int32(m.config.MaxStreams) {
			log.Printf("[mux] reject stream %d: max streams reached (%d)", streamID, m.config.MaxStreams)
			m.sendFrame(cmdClose, streamID, nil)
			return
		}
		
		s := newStream(streamID, m)
		m.streams.Store(streamID, s)
		m.streamCount.Add(1)
		m.totalStreams.Add(1)
		
		log.Printf("[mux] accepted stream %d (total: %d)", streamID, m.streamCount.Load())
		
		select {
		case m.acceptCh <- s:
		case <-m.closeCh:
			return
		case <-time.After(5 * time.Second):
			log.Printf("[mux] accept queue full, dropping stream %d", streamID)
			s.Close()
		}

	case cmdData:
		if val, ok := m.streams.Load(streamID); ok {
			s := val.(*Stream)
			if !s.closed.Load() {
				m.bytesRecv.Add(uint64(len(payload)))
				s.bytesRecv.Add(uint64(len(payload)))
				
				p := make([]byte, len(payload))
				copy(p, payload)
				
				select {
				case s.readCh <- p:
				case <-s.doneCh:
				case <-m.closeCh:
					return
				case <-time.After(m.config.WriteTimeout):
					log.Printf("[mux] stream %d read buffer full, dropping packet", streamID)
				}
			}
		}

	case cmdClose:
		if val, ok := m.streams.Load(streamID); ok {
			s := val.(*Stream)
			s.markClosed()
			m.streams.Delete(streamID)
			m.streamCount.Add(-1)
			log.Printf("[mux] stream %d closed by remote (remaining: %d)", streamID, m.streamCount.Load())
		}

	case cmdPing:
		m.sendFrame(cmdPong, streamID, nil)

	case cmdPong:
	}
}

func (m *Mux) keepAlive() {
	for {
		jitter := time.Duration(randomMuxInt(10000, 15000)) * time.Millisecond
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
	
	if cmd == cmdData {
		m.bytesSent.Add(uint64(len(payload)))
	}

	return m.sc.Send(frame)
}

func (m *Mux) OpenStream() (*Stream, error) {
	return m.OpenStreamWithTimeout(m.config.OpenTimeout)
}

func (m *Mux) OpenStreamWithTimeout(timeout time.Duration) (*Stream, error) {
	select {
	case <-m.closeCh:
		return nil, fmt.Errorf("mux: closed")
	default:
	}
	
	if m.streamCount.Load() >= int32(m.config.MaxStreams) {
		return nil, fmt.Errorf("mux: max streams reached (%d)", m.config.MaxStreams)
	}

	id := m.nextID.Add(1)
	s := newStreamWithConfig(id, m, m.config)
	m.streams.Store(id, s)
	m.streamCount.Add(1)
	m.totalStreams.Add(1)

	if err := m.sendFrame(cmdOpen, id, nil); err != nil {
		m.streams.Delete(id)
		m.streamCount.Add(-1)
		return nil, fmt.Errorf("mux: open: %w", err)
	}

	log.Printf("[mux] opened stream %d (total: %d)", id, m.streamCount.Load())
	return s, nil
}

func (m *Mux) AcceptStream() (*Stream, error) {
	return m.AcceptStreamWithTimeout(m.config.AcceptTimeout)
}

func (m *Mux) AcceptStreamWithTimeout(timeout time.Duration) (*Stream, error) {
	select {
	case s, ok := <-m.acceptCh:
		if !ok {
			return nil, fmt.Errorf("mux: closed")
		}
		return s, nil
	case <-m.closeCh:
		return nil, fmt.Errorf("mux: closed")
	case <-time.After(timeout):
		return nil, fmt.Errorf("mux: accept timeout")
	}
}

func (m *Mux) Close() error {
	m.closeOnce.Do(func() {
		log.Printf("[mux] closing (active streams: %d)", m.streamCount.Load())
		close(m.closeCh)
		
		m.streams.Range(func(key, val any) bool {
			s := val.(*Stream)
			s.markClosed()
			m.streams.Delete(key)
			return true
		})
		
		m.streamCount.Store(0)
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

func (m *Mux) Stats() MuxStats {
	return MuxStats{
		ActiveStreams: int(m.streamCount.Load()),
		TotalStreams:  m.totalStreams.Load(),
		BytesSent:     m.bytesSent.Load(),
		BytesRecv:     m.bytesRecv.Load(),
	}
}

type MuxStats struct {
	ActiveStreams int
	TotalStreams  uint64
	BytesSent     uint64
	BytesRecv     uint64
}

func (m *Mux) ListStreams() []uint32 {
	var ids []uint32
	m.streams.Range(func(key, val any) bool {
		ids = append(ids, key.(uint32))
		return true
	})
	return ids
}

type Stream struct {
	id     uint32
	mux    *Mux
	readCh chan []byte
	doneCh chan struct{}
	closed atomic.Bool

	readMu   sync.Mutex
	readBuf  []byte
	doneOnce sync.Once
	
	bytesSent    atomic.Uint64
	bytesRecv    atomic.Uint64
	created      time.Time
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func newStream(id uint32, m *Mux) *Stream {
	return newStreamWithConfig(id, m, m.config)
}

func newStreamWithConfig(id uint32, m *Mux, cfg *MuxConfig) *Stream {
	return &Stream{
		id:           id,
		mux:          m,
		readCh:       make(chan []byte, cfg.StreamBuffer),
		doneCh:       make(chan struct{}),
		created:      time.Now(),
		readTimeout:  cfg.ReadTimeout,
		writeTimeout: cfg.WriteTimeout,
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

	timeout := time.NewTimer(s.readTimeout)
	defer timeout.Stop()
	
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
	case <-timeout.C:
		return 0, fmt.Errorf("stream: read timeout")
	}
}

func (s *Stream) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}

	const maxChunk = DefaultMaxChunkSize
	sent := 0

	for sent < len(p) {
		end := sent + maxChunk
		if end > len(p) {
			end = len(p)
		}
		
		chunk := p[sent:end]
		if err := s.mux.sendFrame(cmdData, s.id, chunk); err != nil {
			return sent, err
		}
		
		s.bytesSent.Add(uint64(len(chunk)))
		
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
	s.mux.streamCount.Add(-1)
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

func (s *Stream) SetReadDeadline(t time.Time) error {
	if t.IsZero() {
		s.readTimeout = 0
	} else {
		s.readTimeout = time.Until(t)
	}
	return nil
}

func (s *Stream) SetWriteDeadline(t time.Time) error {
	if t.IsZero() {
		s.writeTimeout = 0
	} else {
		s.writeTimeout = time.Until(t)
	}
	return nil
}

func (s *Stream) SetDeadline(t time.Time) error {
	s.SetReadDeadline(t)
	s.SetWriteDeadline(t)
	return nil
}

func (s *Stream) Stats() StreamStats {
	return StreamStats{
		ID:        s.id,
		BytesSent: s.bytesSent.Load(),
		BytesRecv: s.bytesRecv.Load(),
		Age:       time.Since(s.created),
		Closed:    s.closed.Load(),
	}
}

type StreamStats struct {
	ID        uint32
	BytesSent uint64
	BytesRecv uint64
	Age       time.Duration
	Closed    bool
}

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
