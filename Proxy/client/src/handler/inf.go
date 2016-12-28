package handler

import (
	"conn"
	"info"
)

const kProxyBufferSize = 4 * 1024 // 4KB

type IListenHander interface {
	DoIOEvent(sock conn.Socket) *info.HTTPRequest
}

type ISendIOHandler interface {
	DoSendEvent(loaclConn conn.Socket, httpRequest *info.HTTPRequest) error
}
