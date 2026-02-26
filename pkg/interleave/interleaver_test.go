package interleave

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"guarch/pkg/transport"
)

var testPSK = []byte("test-psk-32-bytes-long-key-here!")

func testHandshakeConfig() *transport.HandshakeConfig {
	return &transport.HandshakeConfig{PSK: testPSK}
}

func setupPair(t *testing.T) (*Interleaver, *Interleaver) {
	c1, c2 := net.Pipe()

	var sc1 *transport.SecureConn
	var err1 error
	done := make(chan struct{})

	go func() {
		sc1, err1 = transport.Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	sc2, err2 := transport.Handshake(c2, true, testHandshakeConfig())
	<-done

	if err1 != nil {
		t.Fatal("client handshake:", err1)
	}
	if err2 != nil {
		t.Fatal("server handshake:", err2)
	}

	il1 := New(sc1, nil)
	il2 := New(sc2, nil)

	return il1, il2
}

func TestInterleaverSendRecv(t *testing.T) {
	client, server := setupPair(t)
	defer client.Close()
	defer server.Close()

	msg := []byte("hello through interleaver")

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.SendDirect(msg)
	}()

	data, err := server.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if sendErr := <-errCh; sendErr != nil {
		t.Fatal(sendErr)
	}

	if !bytes.Equal(data, msg) {
		t.Errorf("got %q want %q", data, msg)
	}

	t.Logf("OK: sent and received %d bytes", len(msg))
}

func TestInterleaverBidirectional(t *testing.T) {
	client, server := setupPair(t)
	defer client.Close()
	defer server.Close()

	msg1 := []byte("client to server")
	msg2 := []byte("server to client")

	go func() {
		client.SendDirect(msg1)
	}()

	data1, err := server.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data1, msg1) {
		t.Error("msg1 mismatch")
	}

	go func() {
		server.SendDirect(msg2)
	}()

	data2, err := client.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data2, msg2) {
		t.Error("msg2 mismatch")
	}

	t.Log("OK: bidirectional works")
}

func TestInterleaverPaddingSkipped(t *testing.T) {
	c1, c2 := net.Pipe()

	var sc1 *transport.SecureConn
	var err1 error
	done := make(chan struct{})

	go func() {
		sc1, err1 = transport.Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	sc2, err2 := transport.Handshake(c2, true, testHandshakeConfig())
	<-done

	if err1 != nil || err2 != nil {
		t.Fatal("handshake failed")
	}

	sender := New(sc1, nil)
	receiver := New(sc2, nil)
	defer sender.Close()
	defer receiver.Close()

	go func() {
		sender.sendPadding()
		sender.sendPadding()
		sender.SendDirect([]byte("real data"))
	}()

	data, err := receiver.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, []byte("real data")) {
		t.Errorf("got %q want 'real data'", data)
	}

	t.Log("OK: padding packets skipped, real data received")
}

func TestInterleaverMultiple(t *testing.T) {
	client, server := setupPair(t)
	defer client.Close()
	defer server.Close()

	count := 20
	sendDone := make(chan struct{})

	go func() {
		for i := 0; i < count; i++ {
			client.SendDirect([]byte("message data here"))
		}
		close(sendDone)
	}()

	for i := 0; i < count; i++ {
		_, err := server.Recv()
		if err != nil {
			t.Fatalf("recv %d: %v", i, err)
		}
	}
	<-sendDone

	t.Logf("OK: sent and received %d messages", count)
}

func TestInterleaverWithContext(t *testing.T) {
	c1, c2 := net.Pipe()

	var sc1 *transport.SecureConn
	var err1 error
	done := make(chan struct{})

	go func() {
		sc1, err1 = transport.Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	sc2, err2 := transport.Handshake(c2, true, testHandshakeConfig())
	<-done

	if err1 != nil || err2 != nil {
		t.Fatal("handshake failed")
	}

	sender := New(sc1, nil)
	receiver := New(sc2, nil)
	defer sender.Close()
	defer receiver.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(), 2*time.Second,
	)
	defer cancel()

	sender.Run(ctx)

	go func() {
		sender.SendDirect([]byte("with context"))
	}()

	data, err := receiver.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, []byte("with context")) {
		t.Errorf("got %q", data)
	}

	cancel()

	t.Log("OK: interleaver with context works")
}
