package proxy

import (
	"conn"
	"fmt"
	"handler"
	"net"
	"rule"
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

func (n *nativeProxy) StartListener() {
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
			n.handleConn(conn.CopyConn(netConn))
		}(n, newConn)
	}
}

func (n *nativeProxy) handleConn(netConn *conn.Conn) {
	var err error = nil
	request := handler.NewNativeListener().DoIOEvent(netConn)
	if request == nil {
		fmt.Println("request is null")
		return
	}
	err = handler.NewSendToProxyHandler(n.proxyAddr).DoSendEvent(netConn, request)
	// if n.ruler.IsURLMatched(request.URL) {
	// 	// send to proxy
	// 	fmt.Println("send to proxy")
	// 	err = handler.NewSendToProxyHandler(n.proxyAddr).DoSendEvent(netConn, request)
	// } else {
	// 	fmt.Println("send to native")
	// 	// send to server directlly
	// 	err = handler.NewSendToServerHandler().DoSendEvent(netConn, request)
	// }
	if err != nil {
		fmt.Println("nativeProxy handleRequest error:", err)
	}
}
