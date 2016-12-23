package handler

import (
	"conn"
	"info"
)

const kProxyBufferSize = 4 * 1024 // 4KB

type IListenHander interface {
	DoIOEvent(netConn *conn.Conn) *info.HTTPRequest
}

type ISendIOHandler interface {
	DoSendEvent(loaclConn *conn.Conn, httpRequest *info.HTTPRequest) error
}
