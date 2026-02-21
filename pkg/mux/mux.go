package mux

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"guarch/pkg/transport"
)

const (
	cmdOpen  byte = 0x01
	cmdClose byte = 0x02
	cmdData  byte = 0x03
	cmdPing  byte = 0x04
	cmdPong  byte = 0x05

	muxHeaderSize = 5 // cmd(1) + streamID(4)
)

// ═══════════════════════════════════════
// Mux — مالتی‌پلکسر اتصالات
// ═══════════════════════════════════════

type Mux struct {
	sc        *transport.SecureConn
	streams   sync.Map // map[uint32]*Stream
	nextID    atomic.Uint32
	acceptCh  chan *Stream
	closeCh   chan struct{}
	closeOnce sync.Once
	sendMu    sync.Mutex
}

func NewMux(sc *transport.SecureConn) *Mux {
	m := &Mux{
		sc:       sc,
		acceptCh: make(chan *Stream, 32),
		closeCh:  make(chan struct{}),
	}
	go m.readLoop()
	go m.keepAlive()
	return m
}

// readLoop — تنها goroutine خواننده
func (m *Mux) readLoop() {
	defer m.Close()

	for {
		data, err := m.sc.Recv()
		if err != nil {
			log.Printf("[mux] read loop ended: %v", err)
			return
		}

		if len(data) < muxHeaderSize {
			continue
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
					default:
						log.Printf("[mux] stream %d buffer full", streamID)
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
			m.sendFrame(cmdPong, 0, nil)

		case cmdPong:
			// OK — اتصال زنده است
		}
	}
}

// keepAlive — ارسال Ping دوره‌ای
func (m *Mux) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.closeCh:
			return
		case <-ticker.C:
			if err := m.sendFrame(cmdPing, 0, nil); err != nil {
				return
			}
		}
	}
}

// sendFrame — ارسال فریم مالتی‌پلکس
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

// OpenStream — باز کردن استریم جدید (سمت کلاینت)
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

// AcceptStream — پذیرش استریم (سمت سرور)
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

// Close — بستن مالتی‌پلکسر
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

// IsClosed — آیا مالتی‌پلکسر بسته شده؟
func (m *Mux) IsClosed() bool {
	select {
	case <-m.closeCh:
		return true
	default:
		return false
	}
}

// ═══════════════════════════════════════
// Stream — یک استریم مالتی‌پلکس شده
// ═══════════════════════════════════════

type Stream struct {
	id      uint32
	mux     *Mux
	readCh  chan []byte
	readBuf []byte // بافر خواندن ناقص
	doneCh  chan struct{}
	closed  atomic.Bool
	doneOnce sync.Once
}

func newStream(id uint32, m *Mux) *Stream {
	return &Stream{
		id:     id,
		mux:    m,
		readCh: make(chan []byte, 64),
		doneCh: make(chan struct{}),
	}
}

// Read — خواندن از استریم (io.Reader)
func (s *Stream) Read(p []byte) (int, error) {
	// اول بافر قبلی رو خالی کن
	if len(s.readBuf) > 0 {
		n := copy(p, s.readBuf)
		s.readBuf = s.readBuf[n:]
		return n, nil
	}

	// منتظر داده جدید
	select {
	case data, ok := <-s.readCh:
		if !ok {
			return 0, io.EOF
		}
		n := copy(p, data)
		if n < len(data) {
			// باقیمانده رو بافر کن
			s.readBuf = make([]byte, len(data)-n)
			copy(s.readBuf, data[n:])
		}
		return n, nil
	case <-s.doneCh:
		return 0, io.EOF
	}
}

// Write — نوشتن به استریم (io.Writer)
func (s *Stream) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}

	// تکه‌تکه ارسال کن اگه خیلی بزرگه
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

// Close — بستن استریم
func (s *Stream) Close() error {
	if s.closed.Swap(true) {
		return nil // قبلاً بسته شده
	}

	// به طرف مقابل اطلاع بده
	s.mux.sendFrame(cmdClose, s.id, nil)
	s.mux.streams.Delete(s.id)
	s.markClosed()

	return nil
}

// markClosed — علامت‌گذاری داخلی بسته شدن
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
// RelayStream — ریله بین استریم و اتصال TCP
// ═══════════════════════════════════════

func RelayStream(stream *Stream, conn net.Conn) {
	ch := make(chan error, 2)

	// conn → stream
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

	// stream → conn
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
}
