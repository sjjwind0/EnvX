package conn

import (
	"sync"
)

type connPool struct {
}

var connPoolOnce sync.Once
var connPoolInstance *connPool

func GetConnPool() *connPool {
	connPoolOnce.Do(func() {
		connPoolInstance = new(connPool)
	})
	return connPoolInstance
}

func GetConn() *Conn {
	return nil
}
