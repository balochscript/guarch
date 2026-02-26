package transport

import (
	"bytes"
	"net"
	"testing"

	"guarch/pkg/protocol"
)

var testPSK = []byte("test-psk-32-bytes-long-key-here!")

func testHandshakeConfig() *HandshakeConfig {
	return &HandshakeConfig{PSK: testPSK}
}

func TestHandshake(t *testing.T) {
	c1, c2 := net.Pipe()

	var sc1 *SecureConn
	var err1 error
	done := make(chan struct{})

	go func() {
		sc1, err1 = Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	sc2, err2 := Handshake(c2, true, testHandshakeConfig())
	<-done

	if err1 != nil {
		t.Fatalf("client handshake: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("server handshake: %v", err2)
	}

	_ = sc1
	_ = sc2
	c1.Close()
	c2.Close()

	t.Log("OK: handshake completed")
}

func TestSendRecv(t *testing.T) {
	c1, c2 := net.Pipe()

	var client *SecureConn
	var cerr error
	done := make(chan struct{})

	go func() {
		client, cerr = Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	server, serr := Handshake(c2, true, testHandshakeConfig())
	<-done

	if cerr != nil {
		t.Fatal(cerr)
	}
	if serr != nil {
		t.Fatal(serr)
	}

	msg := []byte("Hello from client to server!")

	sendDone := make(chan struct{})
	go func() {
		if err := client.Send(msg); err != nil {
			t.Errorf("send: %v", err)
		}
		close(sendDone)
	}()

	data, err := server.Recv()
	if err != nil {
		t.Fatal(err)
	}
	<-sendDone

	if !bytes.Equal(data, msg) {
		t.Errorf("got %q want %q", data, msg)
	}

	client.Close()
	server.Close()

	t.Logf("OK: sent and received %d bytes", len(msg))
}

func TestSendRecvBothDirections(t *testing.T) {
	c1, c2 := net.Pipe()

	var client *SecureConn
	var cerr error
	done := make(chan struct{})

	go func() {
		client, cerr = Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	server, serr := Handshake(c2, true, testHandshakeConfig())
	<-done

	if cerr != nil || serr != nil {
		t.Fatal("handshake failed")
	}

	msg1 := []byte("client to server")
	msg2 := []byte("server to client")

	go func() {
		client.Send(msg1)
	}()

	data1, err := server.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data1, msg1) {
		t.Error("msg1 mismatch")
	}

	go func() {
		server.Send(msg2)
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

	t.Log("OK: bidirectional communication works")
}

func TestSendPacket(t *testing.T) {
	c1, c2 := net.Pipe()

	var client *SecureConn
	var cerr error
	done := make(chan struct{})

	go func() {
		client, cerr = Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	server, serr := Handshake(c2, true, testHandshakeConfig())
	<-done

	if cerr != nil || serr != nil {
		t.Fatal("handshake failed")
	}

	pkt, _ := protocol.NewPaddedDataPacket(
		[]byte("padded data"), 100, 200,
	)

	go func() {
		client.SendPacket(pkt)
	}()

	received, err := server.RecvPacket()
	if err != nil {
		t.Fatal(err)
	}

	if received.Type != protocol.PacketTypeData {
		t.Errorf("type: got %s want DATA", received.Type)
	}
	if !bytes.Equal(received.Payload, []byte("padded data")) {
		t.Error("payload mismatch")
	}
	if received.PaddingLen == 0 {
		t.Error("padding lost")
	}

	client.Close()
	server.Close()

	t.Logf("OK: padded packet payload=%d padding=%d",
		received.PayloadLen, received.PaddingLen)
}

func TestMultipleMessages(t *testing.T) {
	c1, c2 := net.Pipe()

	var client *SecureConn
	var cerr error
	done := make(chan struct{})

	go func() {
		client, cerr = Handshake(c1, false, testHandshakeConfig())
		close(done)
	}()

	server, serr := Handshake(c2, true, testHandshakeConfig())
	<-done

	if cerr != nil || serr != nil {
		t.Fatal("handshake failed")
	}

	count := 10
	sendDone := make(chan struct{})

	go func() {
		for i := 0; i < count; i++ {
			msg := []byte("message number test data")
			client.Send(msg)
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
