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
			defer netConn.Close()
			n.handleConn(conn.CopyConn(netConn))
		}(n, newConn)
	}
}

func (n *remoteProxy) handleConn(netConn *conn.Conn) {
	request := handler.NewProxyListener().DoIOEvent(netConn)
	// send to server directlly
	err := handler.NewSendToServerHandler().DoSendEvent(netConn, request)
	if err != nil {
		fmt.Println("remoteProxy handleRequest error:", err)
	}
}
