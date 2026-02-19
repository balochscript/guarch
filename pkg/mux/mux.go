package mux

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"guarch/pkg/transport"
)

const (
	cmdOpen  byte = 0x01
	cmdClose byte = 0x02
	cmdData  byte = 0x03

	muxHeaderSize = 5
)

type Mux struct {
	sc      *transport.SecureConn
	streams sync.Map
	nextID  atomic.Uint32
	mu      sync.Mutex
}

type Stream struct {
	id     uint32
	mux    *Mux
	readCh chan []byte
	closed atomic.Bool
}

func NewMux(sc *transport.SecureConn) *Mux {
	m := &Mux{sc: sc}
	return m
}

func (m *Mux) OpenStream() (*Stream, error) {
	id := m.nextID.Add(1)

	s := &Stream{
		id:     id,
		mux:    m,
		readCh: make(chan []byte, 32),
	}

	m.streams.Store(id, s)

	frame := make([]byte, muxHeaderSize)
	frame[0] = cmdOpen
	binary.BigEndian.PutUint32(frame[1:5], id)

	if err := m.sc.Send(frame); err != nil {
		m.streams.Delete(id)
		return nil, fmt.Errorf("mux: open stream: %w", err)
	}

	log.Printf("[mux] opened stream %d", id)
	return s, nil
}

func (m *Mux) AcceptStream() (*Stream, error) {
	for {
		data, err := m.sc.Recv()
		if err != nil {
			return nil, err
		}

		if len(data) < muxHeaderSize {
			continue
		}

		cmd := data[0]
		streamID := binary.BigEndian.Uint32(data[1:5])

		switch cmd {
		case cmdOpen:
			s := &Stream{
				id:     streamID,
				mux:    m,
				readCh: make(chan []byte, 32),
			}
			m.streams.Store(streamID, s)
			log.Printf("[mux] accepted stream %d", streamID)
			go m.readLoop()
			return s, nil

		case cmdData:
			if val, ok := m.streams.Load(streamID); ok {
				s := val.(*Stream)
				if !s.closed.Load() {
					payload := make([]byte, len(data)-muxHeaderSize)
					copy(payload, data[muxHeaderSize:])
					select {
					case s.readCh <- payload:
					default:
					}
				}
			}

		case cmdClose:
			if val, ok := m.streams.Load(streamID); ok {
				s := val.(*Stream)
				s.closed.Store(true)
				close(s.readCh)
				m.streams.Delete(streamID)
			}
		}
	}
}

func (m *Mux) readLoop() {
	for {
		data, err := m.sc.Recv()
		if err != nil {
			return
		}

		if len(data) < muxHeaderSize {
			continue
		}

		cmd := data[0]
		streamID := binary.BigEndian.Uint32(data[1:5])

		switch cmd {
		case cmdData:
			if val, ok := m.streams.Load(streamID); ok {
				s := val.(*Stream)
				if !s.closed.Load() {
					payload := make([]byte, len(data)-muxHeaderSize)
					copy(payload, data[muxHeaderSize:])
					select {
					case s.readCh <- payload:
					default:
					}
				}
			}

		case cmdClose:
			if val, ok := m.streams.Load(streamID); ok {
				s := val.(*Stream)
				s.closed.Store(true)
				close(s.readCh)
				m.streams.Delete(streamID)
			}
		}
	}
}

func (s *Stream) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}

	frame := make([]byte, muxHeaderSize+len(p))
	frame[0] = cmdData
	binary.BigEndian.PutUint32(frame[1:5], s.id)
	copy(frame[muxHeaderSize:], p)

	if err := s.mux.sc.Send(frame); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (s *Stream) Read(p []byte) (int, error) {
	data, ok := <-s.readCh
	if !ok {
		return 0, io.EOF
	}

	n := copy(p, data)
	return n, nil
}

func (s *Stream) Close() error {
	if s.closed.Swap(true) {
		return nil
	}

	frame := make([]byte, muxHeaderSize)
	frame[0] = cmdClose
	binary.BigEndian.PutUint32(frame[1:5], s.id)
	s.mux.sc.Send(frame)

	s.mux.streams.Delete(s.id)
	return nil
}

func (s *Stream) ID() uint32 {
	return s.id
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
}
