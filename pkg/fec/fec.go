package fec

import (
	"encoding/binary"
)

type FECGroup struct {
	GroupSize int
	packets  [][]byte
	maxLen   int
}

func NewFECGroup(groupSize int) *FECGroup {
	if groupSize < 2 {
		groupSize = 4
	}
	return &FECGroup{
		GroupSize: groupSize,
		packets:  make([][]byte, 0, groupSize),
	}
}

func (fg *FECGroup) Add(data []byte) []byte {
	padded := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(padded[0:2], uint16(len(data)))
	copy(padded[2:], data)

	fg.packets = append(fg.packets, padded)
	if len(padded) > fg.maxLen {
		fg.maxLen = len(padded)
	}

	if len(fg.packets) >= fg.GroupSize {
		fec := fg.generateFEC()
		fg.Reset()
		return fec
	}

	return nil
}

func (fg *FECGroup) generateFEC() []byte {
	result := make([]byte, fg.maxLen)

	for _, pkt := range fg.packets {
		for i := 0; i < len(pkt); i++ {
			result[i] ^= pkt[i]
		}
	}

	return result
}

func (fg *FECGroup) Reset() {
	fg.packets = fg.packets[:0]
	fg.maxLen = 0
}

// ═══════════════════════════════════════
// FEC Decoder
// ═══════════════════════════════════════

type FECDecoder struct {
	GroupSize int
	packets  [][]byte
	fecData  []byte
	received int
}

func NewFECDecoder(groupSize int) *FECDecoder {
	if groupSize < 2 {
		groupSize = 4
	}
	return &FECDecoder{
		GroupSize: groupSize,
		packets:  make([][]byte, groupSize),
	}
}

func (fd *FECDecoder) AddPacket(index int, data []byte) {
	if index < 0 || index >= fd.GroupSize {
		return
	}

	padded := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(padded[0:2], uint16(len(data)))
	copy(padded[2:], data)

	if fd.packets[index] == nil {
		fd.received++
	}
	fd.packets[index] = padded
}

func (fd *FECDecoder) AddFEC(data []byte) {
	fd.fecData = make([]byte, len(data))
	copy(fd.fecData, data)
}

func (fd *FECDecoder) CanRecover() bool {
	if fd.fecData == nil {
		return false
	}
	return fd.received == fd.GroupSize-1
}

func (fd *FECDecoder) Recover() (int, []byte) {
	if !fd.CanRecover() {
		return -1, nil
	}

	missingIdx := -1
	for i, p := range fd.packets {
		if p == nil {
			missingIdx = i
			break
		}
	}
	if missingIdx < 0 {
		return -1, nil
	}

	result := make([]byte, len(fd.fecData))
	copy(result, fd.fecData)

	for i, pkt := range fd.packets {
		if i == missingIdx || pkt == nil {
			continue
		}
		for j := 0; j < len(pkt) && j < len(result); j++ {
			result[j] ^= pkt[j]
		}
	}

	if len(result) < 2 {
		return missingIdx, nil
	}
	origLen := int(binary.BigEndian.Uint16(result[0:2]))
	if origLen+2 > len(result) {
		return missingIdx, result[2:]
	}
	return missingIdx, result[2 : 2+origLen]
}

func (fd *FECDecoder) Reset() {
	for i := range fd.packets {
		fd.packets[i] = nil
	}
	fd.fecData = nil
	fd.received = 0
}
