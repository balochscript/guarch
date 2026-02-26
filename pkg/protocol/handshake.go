package protocol

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	AddrTypeIPv4   byte = 0x01
	AddrTypeDomain byte = 0x03
	AddrTypeIPv6   byte = 0x04

	ConnectSuccess byte = 0x00
	ConnectFailed  byte = 0x01

	MaxDomainLength = 255
)

type ConnectRequest struct {
	AddrType byte
	Addr     string
	Port     uint16
}

func (cr *ConnectRequest) Address() string {
	return fmt.Sprintf("%s:%d", cr.Addr, cr.Port)
}

func (cr *ConnectRequest) Marshal() ([]byte, error) {
	var buf []byte
	buf = append(buf, cr.AddrType)

	switch cr.AddrType {
	case AddrTypeIPv4:
		ip := net.ParseIP(cr.Addr)
		if ip == nil {
			return nil, fmt.Errorf("guarch: invalid IPv4 address: %q", cr.Addr)
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return nil, fmt.Errorf("guarch: address is not IPv4: %q", cr.Addr)
		}
		buf = append(buf, ip4...)

	case AddrTypeDomain:
		if len(cr.Addr) == 0 {
			return nil, fmt.Errorf("guarch: empty domain")
		}
		if len(cr.Addr) > MaxDomainLength {
			return nil, fmt.Errorf("guarch: domain too long: %d (max %d)", len(cr.Addr), MaxDomainLength)
		}
		buf = append(buf, byte(len(cr.Addr)))
		buf = append(buf, []byte(cr.Addr)...)

	case AddrTypeIPv6:
		ip := net.ParseIP(cr.Addr)
		if ip == nil {
			return nil, fmt.Errorf("guarch: invalid IPv6 address: %q", cr.Addr)
		}
		ip16 := ip.To16()
		if ip16 == nil {
			return nil, fmt.Errorf("guarch: address is not IPv6: %q", cr.Addr)
		}
		buf = append(buf, ip16...)

	default:
		return nil, fmt.Errorf("guarch: unknown addr type: %d", cr.AddrType)
	}

	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, cr.Port)
	buf = append(buf, portBuf...)

	return buf, nil
}

func UnmarshalConnectRequest(data []byte) (*ConnectRequest, error) {
	if len(data) < 3 {
		return nil, fmt.Errorf("guarch: connect request too short")
	}

	cr := &ConnectRequest{AddrType: data[0]}
	pos := 1

	switch cr.AddrType {
	case AddrTypeIPv4:
		if len(data) < pos+4+2 {
			return nil, fmt.Errorf("guarch: ipv4 data too short")
		}
		cr.Addr = net.IP(data[pos : pos+4]).String()
		pos += 4
	case AddrTypeDomain:
		dlen := int(data[pos])
		pos++
		if len(data) < pos+dlen+2 {
			return nil, fmt.Errorf("guarch: domain data too short")
		}
		cr.Addr = string(data[pos : pos+dlen])
		pos += dlen
	case AddrTypeIPv6:
		if len(data) < pos+16+2 {
			return nil, fmt.Errorf("guarch: ipv6 data too short")
		}
		cr.Addr = net.IP(data[pos : pos+16]).String()
		pos += 16
	default:
		return nil, fmt.Errorf("guarch: unknown addr type: %d", cr.AddrType)
	}

	cr.Port = binary.BigEndian.Uint16(data[pos : pos+2])
	return cr, nil
}

type ConnectResponse struct {
	Status byte
}

func (cr *ConnectResponse) Marshal() []byte {
	return []byte{cr.Status}
}

func UnmarshalConnectResponse(data []byte) (*ConnectResponse, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("guarch: connect response too short")
	}
	return &ConnectResponse{Status: data[0]}, nil
}
