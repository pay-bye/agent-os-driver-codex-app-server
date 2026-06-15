package control

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

type Connection struct {
	socket net.Conn
	reader *bufio.Reader
}

func NewConnection(socket net.Conn) *Connection {
	return &Connection{socket: socket, reader: bufio.NewReader(socket)}
}

func (c *Connection) Upgrade(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key, err := newUpgradeKey()
	if err != nil {
		return err
	}
	request := "GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := io.WriteString(c.socket, request); err != nil {
		return err
	}
	response, err := http.ReadResponse(c.reader, nil)
	if err != nil {
		return fmt.Errorf("control_upgrade_failed: %w", err)
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("control_upgrade_failed: status %d", response.StatusCode)
	}
	if response.Header.Get("Sec-WebSocket-Accept") != acceptKey(key) {
		return errors.New("control_upgrade_failed: accept key mismatch")
	}
	return nil
}

func (c *Connection) WriteJSON(value any) error {
	content, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.writeText(content)
}

func (c *Connection) writeText(payload []byte) error {
	mask, err := maskingKey()
	if err != nil {
		return err
	}
	header := []byte{0x81}
	header = appendLength(header, len(payload), true)
	frame := append(header, mask[:]...)
	masked := make([]byte, len(payload))
	for index, value := range payload {
		masked[index] = value ^ mask[index%4]
	}
	frame = append(frame, masked...)
	_, err = c.socket.Write(frame)
	return err
}

func (c *Connection) ReadJSON(value any) error {
	content, err := c.readText()
	if err != nil {
		return err
	}
	return json.Unmarshal(content, value)
}

func (c *Connection) readText() ([]byte, error) {
	opcode, payload, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	if opcode != 1 {
		return nil, fmt.Errorf("app_frame_unexpected_opcode: %d", opcode)
	}
	return payload, nil
}

func (c *Connection) readFrame() (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return 0, nil, err
	}
	opcode := header[0] & 0x0f
	length, err := c.frameLength(header[1])
	if err != nil {
		return 0, nil, err
	}
	mask, err := c.frameMask(header[1])
	if err != nil {
		return 0, nil, err
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return 0, nil, err
	}
	if mask != nil {
		for index := range payload {
			payload[index] ^= mask[index%4]
		}
	}
	return opcode, payload, nil
}

func (c *Connection) frameLength(second byte) (int, error) {
	length := int(second & 0x7f)
	if length < 126 {
		return length, nil
	}
	size := 2
	if length == 127 {
		size = 8
	}
	extended := make([]byte, size)
	if _, err := io.ReadFull(c.reader, extended); err != nil {
		return 0, err
	}
	if size == 2 {
		return int(binary.BigEndian.Uint16(extended)), nil
	}
	value := binary.BigEndian.Uint64(extended)
	if value > uint64(int(^uint(0)>>1)) {
		return 0, errors.New("app_frame_too_large")
	}
	return int(value), nil
}

func (c *Connection) frameMask(second byte) ([]byte, error) {
	if second&0x80 == 0 {
		return nil, nil
	}
	mask := make([]byte, 4)
	_, err := io.ReadFull(c.reader, mask)
	return mask, err
}

func (c *Connection) Close() error {
	return c.socket.Close()
}

func newUpgradeKey() (string, error) {
	var key [16]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

func acceptKey(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func maskingKey() ([4]byte, error) {
	var key [4]byte
	_, err := rand.Read(key[:])
	return key, err
}

func appendLength(header []byte, length int, masked bool) []byte {
	prefix := byte(0)
	if masked {
		prefix = 0x80
	}
	if length < 126 {
		return append(header, prefix|byte(length))
	}
	if length <= 0xffff {
		header = append(header, prefix|126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
		return header
	}
	header = append(header, prefix|127, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	return header
}
