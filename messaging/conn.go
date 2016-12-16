package messaging

import (
	"bufio"
	"net"
	"strconv"
	"strings"
)

type MessagesConn struct {
	Conn       net.Conn
	readWriter *bufio.ReadWriter
}

func NewMessagesConn(conn net.Conn) *MessagesConn {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	readWriter := bufio.NewReadWriter(reader, writer)
	messagesConn := &MessagesConn{conn, readWriter}

	return messagesConn
}

func (conn *MessagesConn) Read() ([]byte, error) {
	lenStr, err := conn.readWriter.ReadString('$')
	if err != nil {
		return nil, err
	}

	lenStr = strings.TrimRight(lenStr, "$")
	payloadLen, err := strconv.Atoi(lenStr)
	if err != nil {
		return nil, err
	}

	payload := make([]byte, 0)
	for {
		buffer := make([]byte, payloadLen-len(payload))
		n, err := conn.readWriter.Read(buffer)
		if err != nil {
			return nil, err
		}

		payload = append(payload, buffer[:n]...)

		if len(payload) == payloadLen {
			break
		}
	}

	return payload, nil
}

func (conn *MessagesConn) Write(buffer []byte) (int, error) {
	newBuffer := make([]byte, 0)
	newBuffer = strconv.AppendInt(newBuffer, int64(len(buffer)), 10)
	newBuffer = append(newBuffer, "$"...)
	newBuffer = append(newBuffer, buffer...)

	n, err := conn.readWriter.Write(newBuffer)
	conn.readWriter.Flush()
	return n, err
}

func (conn *MessagesConn) Close() {
	conn.Conn.Close()
}
