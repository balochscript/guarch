package mux

import (
	"bytes"
	"net"
	"testing"

	"guarch/pkg/transport"
)

func setupMux(t *testing.T) (*Mux, *Mux) {
	c1, c2 := net.Pipe()

	var sc1 *transport.SecureConn
	var err1 error
	done := make(chan struct{})

	go func() {
		sc1, err1 = transport.Handshake(c1, false)
		close(done)
	}()

	sc2, err2 := transport.Handshake(c2, true)
	<-done

	if err1 != nil {
		t.Fatal("client handshake:", err1)
	}
	if err2 != nil {
		t.Fatal("server handshake:", err2)
	}

	clientMux := NewMux(sc1)
	serverMux := NewMux(sc2)

	return clientMux, serverMux
}

func TestMuxOpenStream(t *testing.T) {
	clientMux, serverMux := setupMux(t)

	streamDone := make(chan *Stream, 1)
	go func() {
		s, err := serverMux.AcceptStream()
		if err != nil {
			t.Errorf("accept: %v", err)
			streamDone <- nil
			return
		}
		streamDone <- s
	}()

	clientStream, err := clientMux.OpenStream()
	if err != nil {
		t.Fatal(err)
	}

	serverStream := <-streamDone
	if serverStream == nil {
		t.Fatal("server stream is nil")
	}

	if clientStream.ID() == 0 {
		t.Error("client stream ID should not be 0")
	}

	clientStream.Close()
	serverStream.Close()

	t.Logf("OK: stream opened, client=%d server=%d",
		clientStream.ID(), serverStream.ID())
}

func TestMuxWriteRead(t *testing.T) {
	clientMux, serverMux := setupMux(t)

	streamDone := make(chan *Stream, 1)
	go func() {
		s, _ := serverMux.AcceptStream()
		streamDone <- s
	}()

	clientStream, err := clientMux.OpenStream()
	if err != nil {
		t.Fatal(err)
	}

	serverStream := <-streamDone

	msg := []byte("hello through mux stream")

	writeDone := make(chan error, 1)
	go func() {
		_, err := clientStream.Write(msg)
		writeDone <- err
	}()

	buf := make([]byte, 1024)
	n, err := serverStream.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if werr := <-writeDone; werr != nil {
		t.Fatal(werr)
	}

	if !bytes.Equal(buf[:n], msg) {
		t.Errorf("got %q want %q", buf[:n], msg)
	}

	clientStream.Close()
	serverStream.Close()

	t.Logf("OK: wrote and read %d bytes through mux", n)
}

func TestMuxStreamClose(t *testing.T) {
	clientMux, serverMux := setupMux(t)

	streamDone := make(chan *Stream, 1)
	go func() {
		s, _ := serverMux.AcceptStream()
		streamDone <- s
	}()

	clientStream, _ := clientMux.OpenStream()
	serverStream := <-streamDone

	clientStream.Close()

	err := clientStream.Close()
	if err != nil {
		t.Errorf("double close should not error: %v", err)
	}

	serverStream.Close()

	t.Log("OK: stream close works, double close safe")
}

func TestMuxStreamID(t *testing.T) {
	c1, c2 := net.Pipe()

	var sc1 *transport.SecureConn
	var err1 error
	done := make(chan struct{})

	go func() {
		sc1, err1 = transport.Handshake(c1, false)
		close(done)
	}()

	sc2, _ := transport.Handshake(c2, true)
	<-done

	if err1 != nil {
		t.Fatal(err1)
	}

	_ = sc2

	m := NewMux(sc1)

	if m.nextID.Load() != 0 {
		t.Error("initial ID should be 0")
	}

	t.Log("OK: mux created with initial state")
}
