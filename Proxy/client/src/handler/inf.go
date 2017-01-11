package handler

import (
	"conn/socket"
	"info"
)

const kProxyBufferSize = 4 * 1024 // 4KB

type IListenHander interface {
	DoIOEvent(sock socket.Socket) *info.HTTPRequest
}

type ISendIOHandler interface {
	DoSendEvent(loaclConn socket.Socket, httpRequest *info.HTTPRequest) error
}
