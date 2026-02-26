package protocol

import (
	"bytes"
	"testing"
)

func TestNewDataPacket(t *testing.T) {
	payload := []byte("Hello Guarch!")
	pkt, err := NewDataPacket(payload, 1)
	if err != nil {
		t.Fatal(err)
	}

	if pkt.Version != ProtocolVersion {
		t.Errorf("version = %d want %d", pkt.Version, ProtocolVersion)
	}
	if pkt.Type != PacketTypeData {
		t.Errorf("type = %s want DATA", pkt.Type)
	}
	if pkt.SeqNum != 1 {
		t.Errorf("seq = %d want 1", pkt.SeqNum)
	}
	if !bytes.Equal(pkt.Payload, payload) {
		t.Error("payload mismatch")
	}

	t.Logf("OK: %s", pkt)
}

func TestMarshalUnmarshal(t *testing.T) {
	original, err := NewDataPacket([]byte("test data 12345"), 42)
	if err != nil {
		t.Fatal(err)
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if restored.Version != original.Version {
		t.Errorf("version: got %d want %d", restored.Version, original.Version)
	}
	if restored.Type != original.Type {
		t.Errorf("type: got %s want %s", restored.Type, original.Type)
	}
	if restored.SeqNum != original.SeqNum {
		t.Errorf("seq: got %d want %d", restored.SeqNum, original.SeqNum)
	}
	if !bytes.Equal(restored.Payload, original.Payload) {
		t.Error("payload mismatch")
	}

	t.Logf("OK: marshal %d bytes", len(data))
}

func TestPaddedPacket(t *testing.T) {
	pkt, err := NewPaddedDataPacket([]byte("small"), 1, 200)
	if err != nil {
		t.Fatal(err)
	}

	if pkt.TotalSize() < 200 {
		t.Errorf("size = %d want >= 200", pkt.TotalSize())
	}
	if pkt.PaddingLen == 0 {
		t.Error("expected padding")
	}

	data, err := pkt.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored.Payload, []byte("small")) {
		t.Error("payload changed after padding roundtrip")
	}

	t.Logf("OK: payload=%d padding=%d total=%d",
		pkt.PayloadLen, pkt.PaddingLen, pkt.TotalSize())
}

func TestPaddingPacket(t *testing.T) {
	pkt, err := NewPaddingPacket(100, 1)
	if err != nil {
		t.Fatal(err)
	}

	if pkt.Type != PacketTypePadding {
		t.Errorf("type = %s want PADDING", pkt.Type)
	}
	if pkt.PayloadLen != 0 {
		t.Error("payload should be empty")
	}

	data, _ := pkt.Marshal()
	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Type != PacketTypePadding {
		t.Error("type mismatch after roundtrip")
	}

	t.Logf("OK: padding packet %d bytes", pkt.PaddingLen)
}

func TestReadPacketFromStream(t *testing.T) {
	var buf bytes.Buffer

	for i := uint32(1); i <= 5; i++ {
		pkt, _ := NewDataPacket(
			[]byte("packet data here"), i,
		)
		data, _ := pkt.Marshal()
		buf.Write(data)
	}

	for i := uint32(1); i <= 5; i++ {
		pkt, err := ReadPacket(&buf)
		if err != nil {
			t.Fatalf("packet %d: %v", i, err)
		}
		if pkt.SeqNum != i {
			t.Errorf("seq: got %d want %d", pkt.SeqNum, i)
		}
	}

	t.Log("OK: read 5 packets from stream")
}

func TestPingPong(t *testing.T) {
	ping := NewPingPacket(10)
	pong := NewPongPacket(10)

	if ping.Type != PacketTypePing {
		t.Error("ping type wrong")
	}
	if pong.Type != PacketTypePong {
		t.Error("pong type wrong")
	}

	data, _ := ping.Marshal()
	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Type != PacketTypePing {
		t.Error("ping type lost after roundtrip")
	}

	t.Log("OK: ping pong packets work")
}

func TestClosePacket(t *testing.T) {
	pkt := NewClosePacket(99)

	data, _ := pkt.Marshal()
	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Type != PacketTypeClose {
		t.Error("close type lost")
	}
	if restored.SeqNum != 99 {
		t.Error("seq lost")
	}

	t.Log("OK: close packet works")
}

func TestInvalidPacket(t *testing.T) {
	_, err := Unmarshal([]byte{0x01, 0x02})
	if err == nil {
		t.Error("expected error for short packet")
	}

	bad := make([]byte, HeaderSize)
	bad[0] = 0xFF
	bad[1] = 0x01
	_, err = Unmarshal(bad)
	if err == nil {
		t.Error("expected error for wrong version")
	}

	t.Logf("OK: invalid packets rejected")
}
