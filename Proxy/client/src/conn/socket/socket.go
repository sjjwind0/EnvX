package socket

import (
	"errors"
	"fmt"
	"io"
	"net"
)

type TCPSocket struct {
	conn net.Conn
	addr string
}

func NewClientTCPSocket(addr string) *TCPSocket {
	return &TCPSocket{addr: addr}
}

func NewServerTCPSocket(conn net.Conn) *TCPSocket {
	return &TCPSocket{conn: conn}
}

func (t *TCPSocket) Addr() string {
	return t.addr
}

func (t *TCPSocket) Connect() error {
	if t.conn != nil {
		return errors.New("has connected")
	}
	var err error = nil
	t.conn, err = net.Dial("tcp", t.addr)
	return err
}

func (t *TCPSocket) WaitingConnect() error {
	return nil
}

func (t *TCPSocket) Write(data []byte) (int, error) {
	var totalWriteSize int = 0
	var dataSize int = len(data)
	for {
		writeSize, err := t.conn.Write(data)
		if err != nil && err != io.EOF {
			fmt.Println("TCPSocket write failed:", err)
			return totalWriteSize, err
		}
		totalWriteSize += writeSize
		if totalWriteSize == dataSize {
			break
		}
		if err == io.EOF {
			break
		}
	}
	return totalWriteSize, nil
}

func (t *TCPSocket) Read(data []byte) (int, error) {
	return t.conn.Read(data)
}

func (t *TCPSocket) Close() {
	t.conn.Close()
}
