package socks5

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const handshakeTimeout = 30 * time.Second

const maxNMethods = 255

func Handshake(conn net.Conn) (string, error) {
	conn.SetDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetDeadline(time.Time{})

	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", fmt.Errorf("socks5: read greeting: %w", err)
	}

	if buf[0] != 0x05 {
		return "", fmt.Errorf("socks5: invalid version: %d", buf[0])
	}

	nMethods := buf[1]
	if nMethods == 0 {
		return "", fmt.Errorf("socks5: no authentication methods offered")
	}

	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", fmt.Errorf("socks5: read methods: %w", err)
	}

	hasNoAuth := false
	for _, m := range methods {
		if m == 0x00 {
			hasNoAuth = true
			break
		}
	}
	if !hasNoAuth {
		conn.Write([]byte{0x05, 0xFF})
		return "", fmt.Errorf("socks5: no acceptable auth method (need 0x00)")
	}

	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return "", fmt.Errorf("socks5: write auth reply: %w", err)
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", fmt.Errorf("socks5: read request: %w", err)
	}

	if header[0] != 0x05 {
		return "", fmt.Errorf("socks5: invalid version in request")
	}
	if header[1] != 0x01 {
		SendReply(conn, 0x07)
		return "", fmt.Errorf("socks5: only CONNECT supported (got %d)", header[1])
	}

	atyp := header[3]
	var addr string

	switch atyp {
	case 0x01: // IPv4
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("socks5: read ipv4: %w", err)
		}
		addr = net.IP(ip).String()

	case 0x03: // Domain
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", fmt.Errorf("socks5: read domain length: %w", err)
		}
		if lenBuf[0] == 0 {
			return "", fmt.Errorf("socks5: empty domain")
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", fmt.Errorf("socks5: read domain: %w", err)
		}
		addr = string(domain)

	case 0x04: // IPv6
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("socks5: read ipv6: %w", err)
		}
		addr = net.IP(ip).String()

	default:
		SendReply(conn, 0x08)
		return "", fmt.Errorf("socks5: unsupported addr type: 0x%02x", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", fmt.Errorf("socks5: read port: %w", err)
	}
	port := binary.BigEndian.Uint16(portBuf)

	if port == 0 {
		return "", fmt.Errorf("socks5: port 0 is not valid")
	}

	return fmt.Sprintf("%s:%d", addr, port), nil
}

func SendReply(conn net.Conn, status byte) error {
	reply := []byte{
		0x05, status, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00,
	}
	_, err := conn.Write(reply)
	return err
}
