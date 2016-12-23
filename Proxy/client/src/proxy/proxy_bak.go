package proxy

import (
	"conn"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
)

const (
	kProxyBufferSize = 4096 // 4kb buffer
)

// Proxy returns a Dialer that makes SOCKSv5 connections to the given address
// with an optional username and password. See RFC 1928.
func NewProxy(ip, port string) *proxy {
	p := &proxy{
		ip:   ip,
		port: port,
	}

	return p
}

const kHTTPRequest = 0x0
const kHTTPSRequest = 0x1

// TODO: impl tcp、udp、ftp request
const kTCPRequest = 0x2
const KUDPRequest = 0x3

type HTTPRequest struct {
	Addr            string   `json:"addr"`
	Method          string   `json:"method"`
	URL             string   `json:"url"`
	ProtocolVersion string   `json:"version"`
	Header          []string `json:"header"`
	Body            string   `json:"body"`
}

type proxy struct {
	ip   string
	port string
}

func (p *proxy) StartProxyListener(localIp string, localPort string) {
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", localIp+":"+localPort)
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
		go func(p *proxy, netConn net.Conn) {
			defer netConn.Close()
			p.handleRequest(conn.CopyConn(netConn))
		}(p, c)
	}
}

func (p *proxy) handleRequest(c *conn.Conn) {
	// TODO: find free tcp channel from tcp channel pool.
	proxyConn, err := conn.NewTCPConn(p.ip + ":" + p.port)
	if err != nil {
		fmt.Println("connect proxy server error: ", err)
		return
	}
	var readBuf []byte = make([]byte, kProxyBufferSize)
	var totalBuf []byte = nil
	var totalSize int = 0
	var hasRecvHeader bool = false
	var headerEndIndex int = 0
	/* read first \r\n\r\n segment，if current request is https, we will get addr info from it.
	 * if current request is http, we will get http header.
	 */
	for true {
		readSize, err := c.Read(readBuf)
		if err == io.EOF {
			fmt.Println("readSize: ", readSize)
			break
		}
		totalBuf = append(totalBuf, readBuf...)
		totalSize += readSize
		// get \r\n\r\n from data.
		for i := 0; i+3 < totalSize; i++ {
			if totalBuf[i] == 13 && totalBuf[i+1] == 10 &&
				totalBuf[i+2] == 13 && totalBuf[i+3] == 10 {
				hasRecvHeader = true
				headerEndIndex = i + 4
				break
			}
		}
		if hasRecvHeader {
			break
		}
	}
	if totalSize == 0 {
		return
	}
	var bodyBuffer []byte = nil
	if headerEndIndex < totalSize {
		bodyBuffer = totalBuf[headerEndIndex:]
	}
	// parse content to find https source addr info or http real request.
	var header string = string(totalBuf[:totalSize])
	headerLines := strings.Split(header, "\r\n")
	firstLineContents := strings.Split(headerLines[0], " ")
	var request HTTPRequest
	request.Method = firstLineContents[0]
	request.URL = firstLineContents[1]
	request.Addr = request.URL

	var port string = "80"
	var beginFindIndex int = 0
	if strings.HasPrefix(request.URL, "http://") {
		beginFindIndex = 7
	}
	if strings.HasPrefix(request.URL, "https://") {
		beginFindIndex = 8
	}
	findIndex := strings.Index(request.URL[beginFindIndex:], ":")
	if findIndex != -1 {
		secondIndex1 := strings.Index(request.URL[findIndex:], "/") + findIndex
		secondIndex2 := strings.Index(request.URL[findIndex:], "?") + findIndex
		secondIndex := secondIndex1
		if secondIndex > secondIndex2 {
			secondIndex = secondIndex2
		}
		if secondIndex == -1 {
			secondIndex = len(request.URL)
		}
		fmt.Println("url: ", request.URL)
		port = request.URL[findIndex+1 : secondIndex]
	}

	request.ProtocolVersion = firstLineContents[2]
	for i := 1; i < len(headerLines); i++ {
		if len(headerLines[i]) > 1 {
			request.Header = append(request.Header, headerLines[i])
			if strings.HasPrefix(headerLines[i], "Host: ") {
				addr := headerLines[i][6:]
				if strings.Index(addr, ":") == -1 {
					request.Addr = addr + ":" + port
				} else {
					request.Addr = addr
				}
			}
		}
	}
	p.startDialProxyServer(c, proxyConn, &request, bodyBuffer)
}

func (p *proxy) startDialProxyServer(localConn *conn.Conn, proxyConn *conn.Conn,
	request *HTTPRequest, bodyBuffer []byte) error {
	data, err := json.Marshal(request)
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
	if bodyBuffer != nil {
		proxyConn.Write(bodyBuffer)
	}

	go conn.Copy(proxyConn, localConn)
	conn.Copy(localConn, proxyConn)
	return nil
}
