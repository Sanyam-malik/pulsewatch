package executor

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

const a2sInfoRequest = "\xff\xff\xff\xffTSource Engine Query\x00"

type a2sInfo struct {
	Name        string
	Map         string
	Folder      string
	Game        string
	AppID       uint16
	Players     uint8
	MaxPlayers  uint8
	Bots        uint8
	ServerType  byte
	Environment byte
	Password    bool
	VAC         bool
	Version     string
}

func queryA2SInfo(ctx context.Context, host string, port int) (*a2sInfo, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := (&net.Dialer{}).DialContext(ctx, "udp", address)
	if err != nil {
		return nil, fmt.Errorf("connect to game server: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	query := []byte(a2sInfoRequest)
	response, err := exchangeA2SPacket(conn, query)
	if err != nil {
		return nil, err
	}
	if len(response) >= 9 && response[4] == 'A' {
		query = append(query, response[5:9]...)
		response, err = exchangeA2SPacket(conn, query)
		if err != nil {
			return nil, err
		}
	}

	info, err := parseA2SInfo(response)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func exchangeA2SPacket(conn net.Conn, query []byte) ([]byte, error) {
	if _, err := conn.Write(query); err != nil {
		return nil, fmt.Errorf("send A2S query: %w", err)
	}

	response := make([]byte, 65535)
	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("read A2S response: %w", err)
	}
	return response[:n], nil
}

func parseA2SInfo(packet []byte) (*a2sInfo, error) {
	if len(packet) < 5 || binary.LittleEndian.Uint32(packet[:4]) != ^uint32(0) {
		return nil, fmt.Errorf("invalid A2S response header")
	}
	if packet[4] != 'I' {
		if packet[4] == 'A' {
			return nil, fmt.Errorf("server returned an incomplete A2S challenge")
		}
		return nil, fmt.Errorf("unexpected A2S response type %q", packet[4])
	}

	reader := a2sReader{data: packet, offset: 5}
	protocol, err := reader.byte()
	if err != nil {
		return nil, err
	}
	info := &a2sInfo{}
	if info.Name, err = reader.string(); err != nil {
		return nil, err
	}
	if info.Map, err = reader.string(); err != nil {
		return nil, err
	}
	if info.Folder, err = reader.string(); err != nil {
		return nil, err
	}
	if info.Game, err = reader.string(); err != nil {
		return nil, err
	}
	appID, err := reader.uint16()
	if err != nil {
		return nil, err
	}
	info.AppID = appID
	if info.Players, err = reader.byte(); err != nil {
		return nil, err
	}
	if info.MaxPlayers, err = reader.byte(); err != nil {
		return nil, err
	}
	if info.Bots, err = reader.byte(); err != nil {
		return nil, err
	}
	if info.ServerType, err = reader.byte(); err != nil {
		return nil, err
	}
	if info.Environment, err = reader.byte(); err != nil {
		return nil, err
	}
	password, err := reader.byte()
	if err != nil {
		return nil, err
	}
	info.Password = password != 0
	vac, err := reader.byte()
	if err != nil {
		return nil, err
	}
	info.VAC = vac != 0
	if info.Version, err = reader.string(); err != nil {
		return nil, err
	}

	if protocol == 0 || info.Name == "" {
		return nil, fmt.Errorf("A2S response is missing required server information")
	}
	return info, nil
}

type a2sReader struct {
	data   []byte
	offset int
}

func (r *a2sReader) byte() (byte, error) {
	if r.offset >= len(r.data) {
		return 0, fmt.Errorf("truncated A2S info response")
	}
	value := r.data[r.offset]
	r.offset++
	return value, nil
}

func (r *a2sReader) uint16() (uint16, error) {
	if r.offset+2 > len(r.data) {
		return 0, fmt.Errorf("truncated A2S info response")
	}
	value := binary.LittleEndian.Uint16(r.data[r.offset : r.offset+2])
	r.offset += 2
	return value, nil
}

func (r *a2sReader) string() (string, error) {
	start := r.offset
	for r.offset < len(r.data) && r.data[r.offset] != 0 {
		r.offset++
	}
	if r.offset >= len(r.data) {
		return "", fmt.Errorf("unterminated string in A2S info response")
	}
	value := string(r.data[start:r.offset])
	r.offset++
	return value, nil
}
