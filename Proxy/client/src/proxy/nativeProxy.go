package proxy

import (
	"conn"
	"fmt"
	"info"
)

type nativeProxy struct {
	nativeAddr string
	proxyAddr  string
	ruler      *rule.GFWRuleParser
}

func NewNativeProxy(nativeAddr string, proxyAddr string) *nativeProxy {
	return &nativeProxy{
		nativeAddr: nativeAddr,
		proxyAddr:  proxyAddr,
		ruler:      rule.NewGFWRuleParser(),
	}
}

func (n *nativeProxy) startListener() {
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
		go func(n *nativeProxy, netConn net.Conn) {
			defer netConn.Close()
			p.handleConn(conn.CopyConn(netConn))
		}(n, newConn)
	}
}

func (n *nativeProxy) handleConn(netConn *conn.Conn) {
	var err error = nil
	request := handler.NewNativeListener().DoIOEvent(conn.CopyConn(netConn))
	if ruler.IsURLMatch(request.URL) {
		// send to proxy
		err = NewSendToProxyHandler(n.proxyAddr).DoSendEvent(netConn, request)
	} else {
		// send to server directlly
		err = NewSendToServerHandler().DoSendEvent(netConn, request)
	}
	if err != nil {
		fmt.Println("nativeProxy handleRequest error:", err)
	}
}
