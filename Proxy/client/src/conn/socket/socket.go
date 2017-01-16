package socket

import (
	"errors"
	"fmt"
	"io"
	"net"
)

func TCPSocketListen(addr string) (*TCPSocket, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newServerTCPSocket(listener), nil
}

type TCPSocket struct {
	conn           net.Conn
	addr           string
	socketListener net.Listener
}

func NewTCPSocket(addr string) *TCPSocket {
	return &TCPSocket{addr: addr}
}

func newServerTCPSocket(listener net.Listener) *TCPSocket {
	return &TCPSocket{socketListener: listener}
}

func newServerTCPSocketFromNetConn(conn net.Conn) *TCPSocket {
	return &TCPSocket{conn: conn}
}

func (t *TCPSocket) Addr() string {
	return t.addr
}

func (t *TCPSocket) Accept() (Socket, error) {
	acceptConn, err := t.socketListener.Accept()
	if err != nil {
		fmt.Println("SecurityTCPSocket accept error:", err)
		return nil, err
	}
	securityTCPSocket := newServerTCPSocketFromNetConn(acceptConn)
	return securityTCPSocket, nil
}

func (t *TCPSocket) Connect() error {
	if t.conn != nil {
		return errors.New("has connected")
	}
	var err error = nil
	t.conn, err = net.Dial("tcp", t.addr)
	return err
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
