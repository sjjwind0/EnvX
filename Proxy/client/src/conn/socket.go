package conn

import (
	"conn/socket"
	"errors"
	"io"
)

const kCacheBufferSize = 4 * 1024

func Listen(net string, addr string) (socket.Listener, error) {
	if net == "tcp" {
		sock, err := socket.TCPSocketListen(addr)
		return sock, err
	} else if net == "sts" {
		realSock, err := socket.SecurityTCPSocketListen(addr)
		return realSock, err
	}
	return nil, errors.New("unspport net")
}

var globalSocket *socket.RealSecurityTCPSocket = nil

func Dial(net string, addr string) (socket.Socket, error) {
	var clientSocket socket.Socket = nil
	var err error = nil
	if net == "tcp" {
		tcpSocket := socket.NewTCPSocket(addr)
		err = tcpSocket.Connect()
		if err != nil {
			return nil, err
		}
		clientSocket = tcpSocket
	} else if net == "sts" {
		if globalSocket == nil {
			globalSocket = socket.NewRealSecurityTCPSocketWithAddr(addr)
			err = globalSocket.Connect()
			if err != nil {
				return nil, err
			}
		}
		clientSocket = globalSocket.NewClientVirtualSocket()
	}
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
