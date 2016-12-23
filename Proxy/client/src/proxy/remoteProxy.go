package proxy

import (
	"conn"
	"fmt"
	"info"
)

type remoteProxy struct {
	nativeAddr string
	ruler      *rule.GFWRuleParser
}

func NewRemoteProxy(nativeAddr string) *remoteProxy {
	return &remoteProxy{
		nativeAddr: nativeAddr,
		ruler:      rule.NewGFWRuleParser(),
	}
}

func (n *remoteProxy) startListener() {
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
			p.handleConn(conn.CopyConn(netConn))
		}(n, newConn)
	}
}

func (n *remoteProxy) handleConn(netConn *conn.Conn) {
	request := handler.NewProxyListener().DoIOEvent(conn.CopyConn(netConn))
	// send to server directlly
	err := NewSendToServerHandler().DoSendEvent(netConn, request)
	if err != nil {
		fmt.Println("remoteProxy handleRequest error:", err)
	}
}
