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

func (n *nativeProxy) handleConn(sock conn.Socket) {
	var err error = nil
	request := handler.NewNativeListener().DoIOEvent(sock)
	if request == nil {
		fmt.Println("request is null")
		return
	}
	err = handler.NewSendToProxyHandler(n.proxyAddr).DoSendEvent(sock, request)
	// if n.ruler.IsURLMatched(request.URL) {
	// 	// send to proxy
	// 	fmt.Println("send to proxy")
	// 	err = handler.NewSendToProxyHandler(n.proxyAddr).DoSendEvent(sock, request)
	// } else {
	// 	fmt.Println("send to native")
	// 	// send to server directlly
	// 	err = handler.NewSendToServerHandler().DoSendEvent(sock, request)
	// }
	if err != nil {
		fmt.Println("nativeProxy handleRequest error:", err)
	}
}
