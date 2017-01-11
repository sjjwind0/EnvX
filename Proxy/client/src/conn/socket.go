package conn

import (
	"conn/socket"
	"errors"
	"io"
)

const kCacheBufferSize = 4 * 1024

func Listen(net string, addr string) (socket.Socket, error) {
	if net == "tcp" {
		sock, err := socket.TCPSocketListen(addr)
		return sock, err
	} else if net == "sts" {
		sock, err := socket.SecurityTCPSocketListen(addr)
		return sock, err
	}
	return nil, errors.New("unspport net")
}

func Dial(net string, addr string) (socket.Socket, error) {
	var clientSocket socket.Socket = nil
	if net == "tcp" {
		clientSocket = socket.NewTCPSocket(addr)
	} else if net == "sts" {
		clientSocket = socket.NewSecurityTCPSocket(addr)
	}
	err := clientSocket.Connect()
	if err != nil {
		return nil, err
	}
	return clientSocket, err
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
