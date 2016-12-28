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

func (s *sendToProxyHandler) DoSendEvent(localSock conn.Socket, httpRequest *info.HTTPRequest) error {
	proxySock := conn.NewTCPSocket(s.proxyAddr)
	err := proxySock.Connect()
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
	proxySock.Write(headerByte)
	proxySock.Write(data)
	if httpRequest.ExtraData != nil {
		proxySock.Write(httpRequest.ExtraData)
	}

	go conn.Copy(proxySock, localSock)
	conn.Copy(localSock, proxySock)
	return nil
}
