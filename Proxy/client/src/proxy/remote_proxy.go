package proxy

import (
	"conn"
	"conn/socket"
	"fmt"
	"handler"
)

type remoteProxy struct {
	nativeAddr string
}

func NewRemoteProxy(nativeAddr string) *remoteProxy {
	return &remoteProxy{
		nativeAddr: nativeAddr,
	}
}

func (n *remoteProxy) StartListener() {
	var listener socket.Listener
	var err error
	listener, err = conn.Listen("sts", n.nativeAddr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	for {
		acceptSocket, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			return
		}
		go func(n *remoteProxy, acceptSocket socket.Socket) {
			defer acceptSocket.Close()
			n.handleConn(acceptSocket)
		}(n, acceptSocket)
	}
}

func (n *remoteProxy) handleConn(sock socket.Socket) {
	request := handler.NewProxyListener().DoIOEvent(sock)
	// send to server directlly
	err := handler.NewSendToServerHandler().DoSendEvent(sock, request)
	if err != nil {
		fmt.Println("remoteProxy handleRequest error:", err)
	}
}
