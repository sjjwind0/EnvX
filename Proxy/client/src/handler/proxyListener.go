package handler

import (
	"conn"
	"fmt"
	"info"
)

type proxyListener struct {
}

func NewProxyListener() {
	return new(proxyListener)
}

func (n *proxyListener) DoIOEvent(netConn *conn.Conn) *info.HTTPRequest {
	var buf []byte = make([]byte, 4)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("err: ", err)
		return
	}
	dataSize := (((int)(buf[0])) << 24) | (((int)(buf[1])) << 16) | (((int)(buf[2])) << 8) | (int)(buf[3])
	var header []byte = make([]byte, dataSize)
	_, err = conn.Read(header)
	if err != nil {
		fmt.Println("read error: ", err)
		return
	}
	var httpRequest HTTPRequest
	err = json.Unmarshal(header, &httpRequest)
	if err != nil {
		fmt.Println("unmarshal error: ", err)
		return
	}
	return &httpRequest
}
