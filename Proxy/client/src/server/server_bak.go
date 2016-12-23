package server

import (
	"conn"
	"encoding/json"
	"fmt"
	"net"
)

type HTTPRequest struct {
	Addr            string   `json:"addr"`
	Method          string   `json:"method"`
	URL             string   `json:"url"`
	ProtocolVersion string   `json:"version"`
	Header          []string `json:"header"`
	Body            string   `json:"body"`
}

func StartServer(host string, port string) {
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", host+":"+port)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			return
		}
		go func() {
			defer c.Close()
			handleRequest(conn.CopyConn(c))
		}()
	}
}

func handleRequest(conn *conn.Conn) {
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
	doRequest(conn, &httpRequest)
}

func doRequest(conn *conn.Conn, httpRequest *HTTPRequest) error {
	if httpRequest.Method == "CONNECT" {
		return doHTTPSProxyRequest(conn, httpRequest)
	} else {
		return doHTTPRequest(conn, httpRequest)
	}
}

func doHTTPSProxyRequest(nativeConn *conn.Conn, httpRequest *HTTPRequest) error {
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

func doHTTPRequest(nativeConn *conn.Conn, httpRequest *HTTPRequest) error {
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
