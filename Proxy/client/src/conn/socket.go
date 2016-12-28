package conn

import (
	"conn/socket"
	"io"
	"net"
)

const (
	kCacheBufferSize = 4 * 1024 // 4KB
)

type Socket interface {
	Addr() string
	Connect() error
	WaitingConnect() error
	Read(readData []byte) (int, error)
	Write(writeData []byte) (int, error)
	Close()
}

func NewTCPSocket(addr string) Socket {
	return socket.NewClientTCPSocket(addr)
}

func NewSecurityTCPSocket(addr string) Socket {
	return socket.NewSecurityClientTCPSocket(addr)
}

func NewTCPSocketFromConn(conn net.Conn) Socket {
	return socket.NewServerTCPSocket(conn)
}

func NewSecurityTCPSocketFromConn(conn net.Conn) Socket {
	return socket.NewSecurityServerTCPSocket(conn)
}

func Copy(src io.Writer, dst io.Reader) error {
	var readBuffer []byte = make([]byte, kCacheBufferSize)
	for {
		readSize, err := dst.Read(readBuffer)
		if err != nil && err != io.EOF {
			return err
		}
		_, writeErr := src.Write(readBuffer[:readSize])
		if writeErr != nil {
			return writeErr
		}
		if err == io.EOF {
			return nil
		}
	}
}
