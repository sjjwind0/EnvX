package handler

import (
	"conn"
	"encoding/json"
	"fmt"
	"info"
)

type sendToProxyHandler struct {
	proxyAddr string
}

func NewSendToProxyHandler(addr string) *sendToProxyHandler {
	return &sendToProxyHandler{proxyAddr: addr}
}

func (s *sendToProxyHandler) DoSendEvent(loaclConn *conn.Conn, httpRequest *info.HTTPRequest) error {
	proxyConn, err := conn.NewTCPConn(s.proxyAddr)
	if err != nil {
		fmt.Println("connect proxy server error: ", err)
		return err
	}

	data, err := json.Marshal(httpRequest)
	if err != nil {
		fmt.Println("encoding json error: ", err)
		return err
	}
	// TODO: crypto and compress.
	dataSize := len(data)
	headerByte := []byte{
		byte(dataSize >> 24),
		byte((dataSize & 0xFF0000) >> 16),
		byte((dataSize & 0xFF00) >> 8),
		byte(dataSize & 0xFF),
	}
	proxyConn.Write(headerByte)
	proxyConn.Write(data)
	if httpRequest.ExtraData != nil {
		proxyConn.Write(httpRequest.ExtraData)
	}

	go conn.Copy(proxyConn, loaclConn)
	conn.Copy(loaclConn, proxyConn)
	return nil
}
