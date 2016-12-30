package handler

import (
	"conn"
	"fmt"
	"info"
)

type sendToServerHandler struct {
}

func NewSendToServerHandler() *sendToServerHandler {
	return new(sendToServerHandler)
}

func (s *sendToServerHandler) DoSendEvent(loaclSock conn.Socket, httpRequest *info.HTTPRequest) error {
	if httpRequest == nil {
		return nil
	}
	fmt.Println("recv: ", httpRequest)
	if httpRequest.Method == "CONNECT" {
		return s.doHTTPSRequest(loaclSock, httpRequest)
	} else {
		return s.doHTTPRequest(loaclSock, httpRequest)
	}
}

func (s *sendToServerHandler) doHTTPSRequest(nativeSock conn.Socket, httpRequest *info.HTTPRequest) error {
	serverSock := conn.NewTCPSocket(httpRequest.Addr)
	err := serverSock.Connect()
	if err != nil {
		fmt.Println("connect", httpRequest.Addr, "failed:", err)
		fmt.Println("httpRequest: ", httpRequest)
		return err
	}
	// tell client, client success.
	_, err = nativeSock.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		fmt.Println("conn.Write failed: ", err)
		return err
	}
	go conn.Copy(serverSock, nativeSock)
	conn.Copy(nativeSock, serverSock)
	return nil
}

func (s *sendToServerHandler) doHTTPRequest(nativeSock conn.Socket, httpRequest *info.HTTPRequest) error {
	serverSock := conn.NewTCPSocket(httpRequest.Addr)
	err := serverSock.Connect()
	if err != nil {
		fmt.Println("new request error: ", err)
		return err
	}

	var rawHTTPRequest string = ""
	rawHTTPRequest += httpRequest.Method + " "
	rawHTTPRequest += httpRequest.URL + " "
	rawHTTPRequest += httpRequest.ProtocolVersion
	rawHTTPRequest += "\r\n"
	// header
	for _, line := range httpRequest.Header {
		rawHTTPRequest += line + "\r\n"
	}
	rawHTTPRequest += "\r\n"
	fmt.Println("raw: ", rawHTTPRequest)
	// write header info
	go func() {
		_, err = serverSock.Write([]byte(rawHTTPRequest))
		if err != nil {
			fmt.Println("write raw header error: ", err)
			return
		}

		if len(httpRequest.Body) != 0 {
			serverSock.Write([]byte(httpRequest.Body))
		}
		conn.Copy(serverSock, nativeSock)
	}()
	conn.Copy(nativeSock, serverSock)

	return nil
}
