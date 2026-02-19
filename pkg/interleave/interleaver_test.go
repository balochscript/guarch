package interleave

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/transport"
)

func setupPair(t *testing.T) (*Interleaver, *Interleaver) {
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

	il1 := New(sc1, nil)
	il2 := New(sc2, nil)

	return il1, il2
}

func TestInterleaverSendRecv(t *testing.T) {
	client, server := setupPair(t)

	msg := []byte("hello through interleaver")

	go func() {
		client.SendDirect(msg)
	}()

	data, err := server.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, msg) {
		t.Errorf("got %q want %q", data, msg)
	}

	client.Close()
	server.Close()

	t.Logf("OK: sent and received %d bytes", len(msg))
}

func TestInterleaverBidirectional(t *testing.T) {
	client, server := setupPair(t)

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

	client.Close()
	server.Close()

	t.Log("OK: bidirectional works")
}

func TestInterleaverPaddingSkipped(t *testing.T) {
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

	if err1 != nil || err2 != nil {
		t.Fatal("handshake failed")
	}

	sender := New(sc1, nil)
	receiver := New(sc2, nil)

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

	sender.Close()
	receiver.Close()

	t.Log("OK: padding packets skipped, real data received")
}

func TestInterleaverMultiple(t *testing.T) {
	client, server := setupPair(t)

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

	client.Close()
	server.Close()

	t.Logf("OK: sent and received %d messages", count)
}

func TestInterleaverWithCover(t *testing.T) {
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

	if err1 != nil || err2 != nil {
		t.Fatal("handshake failed")
	}

	stats := cover.NewStats(10)
	stats.Record(500)
	stats.Record(500)

	sender := New(sc1, nil)
	receiver := New(sc2, nil)

	ctx, cancel := context.WithTimeout(
		context.Background(), 3*time.Second,
	)
	defer cancel()

	sender.Run(ctx)

	go func() {
		sender.SendDirect([]byte("covered message"))
	}()

	data, err := receiver.Recv()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, []byte("covered message")) {
		t.Errorf("got %q", data)
	}

	cancel()
	sender.Close()
	receiver.Close()

	t.Log("OK: interleaver with cover context works")
}
