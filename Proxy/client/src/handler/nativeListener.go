package handler

import (
	"conn"
	"fmt"
	"info"
	"strings"
)

type nativeListener struct {
}

func NewNativeListener() {
	return new(nativeListener)
}

func (n *nativeListener) DoIOEvent(netConn *conn.Conn) *info.HTTPRequest {
	var readBuf []byte = make([]byte, kProxyBufferSize)
	var totalBuf []byte = nil
	var totalSize int = 0
	var hasRecvHeader bool = false
	var headerEndIndex int = 0
	/* read first \r\n\r\n segment，if current request is https, we will get addr info from it.
	 * if current request is http, we will get http header.
	 */
	for true {
		readSize, err := netConn.Read(readBuf)
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
	return n.buildHTTPRequestFromFullBuffer(totalBuf, headerEndIndex)
}

func (n *nativeListener) buildHTTPRequestFromFullBuffer(totalBuf *[]byte, endIndex int) *info.HTTPRequest {
	var request *info.HTTPRequest = new(info.HTTPRequest)
	if endIndex < totalSize {
		request.ExtraData = totalBuf[endIndex:]
	}
	// parse content to find https source addr info or http real request.
	var header string = string(totalBuf[:totalSize])
	headerLines := strings.Split(header, "\r\n")
	firstLineContents := strings.Split(headerLines[0], " ")
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
	return request
}