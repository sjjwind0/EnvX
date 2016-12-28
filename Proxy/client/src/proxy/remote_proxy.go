package proxy

import (
	"conn"
	"fmt"
	"handler"
	"net"
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
	var listener net.Listener
	var err error
	listener, err = net.Listen("tcp", n.nativeAddr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	for {
		newConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			return
		}
		go func(n *remoteProxy, netConn net.Conn) {
			serverSock := conn.NewTCPSocketFromConn(netConn)
			defer serverSock.Close()
			err := serverSock.WaitingConnect()
			if err != nil {
				fmt.Println("waiting error: ", err)
				return
			}
			n.handleConn(serverSock)
		}(n, newConn)
	}
}

func (n *remoteProxy) handleConn(sock conn.Socket) {
	request := handler.NewProxyListener().DoIOEvent(sock)
	// send to server directlly
	err := handler.NewSendToServerHandler().DoSendEvent(sock, request)
	if err != nil {
		fmt.Println("remoteProxy handleRequest error:", err)
	}
}
