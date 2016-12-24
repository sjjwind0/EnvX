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

func (s *sendToServerHandler) DoSendEvent(loaclConn *conn.Conn, httpRequest *info.HTTPRequest) error {
	if httpRequest.Method == "CONNECT" {
		return s.doHTTPSRequest(loaclConn, httpRequest)
	} else {
		return s.doHTTPRequest(loaclConn, httpRequest)
	}
}

func (s *sendToServerHandler) doHTTPSRequest(nativeConn *conn.Conn, httpRequest *info.HTTPRequest) error {
	serverConn, err := conn.NewTCPConn(httpRequest.Addr)
	if err != nil {
		fmt.Println("connect", httpRequest.Addr, "failed:", err)
		fmt.Println("httpRequest: ", httpRequest)
		return err
	}
	// tell client, client success.
	_, err = nativeConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		fmt.Println("conn.Write failed: ", err)
		return err
	}
	go conn.Copy(serverConn, nativeConn)
	conn.Copy(nativeConn, serverConn)
	return nil
}

func (s *sendToServerHandler) doHTTPRequest(nativeConn *conn.Conn, httpRequest *info.HTTPRequest) error {
	serverConn, err := conn.NewTCPConn(httpRequest.Addr)
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
	// write header info
	go func() {
		_, err = serverConn.Write([]byte(rawHTTPRequest))
		if err != nil {
			fmt.Println("write raw header error: ", err)
			return
		}

		if len(httpRequest.Body) != 0 {
			serverConn.Write([]byte(httpRequest.Body))
		}
		conn.Copy(serverConn, nativeConn)
	}()
	conn.Copy(nativeConn, serverConn)

	return nil
}
