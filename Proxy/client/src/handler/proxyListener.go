package handler

import (
	"conn/socket"
	"encoding/json"
	"fmt"
	"info"
)

type proxyListener struct {
}

func NewProxyListener() *proxyListener {
	return new(proxyListener)
}

func (n *proxyListener) DoIOEvent(sock socket.Socket) *info.HTTPRequest {
	var buf []byte = make([]byte, 4)
	_, err := sock.Read(buf)
	if err != nil {
		fmt.Println("err: ", err)
		return nil
	}
	dataSize := (((int)(buf[0])) << 24) | (((int)(buf[1])) << 16) | (((int)(buf[2])) << 8) | (int)(buf[3])
	var header []byte = make([]byte, dataSize)
	_, err = sock.Read(header)
	if err != nil {
		fmt.Println("read error: ", err)
		return nil
	}
	var httpRequest info.HTTPRequest
	err = json.Unmarshal(header, &httpRequest)
	if err != nil {
		fmt.Println("unmarshal error: ", err)
		return nil
	}
	return &httpRequest
}
