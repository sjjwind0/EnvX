package conn

import (
	"conn/socket"
	"container/list"
	"fmt"
	"sync"
)

const (
	kMaxSocketPoolCount = 20
)

type socketPool struct {
	socketMap map[string]*list.List
	mutex     sync.Mutex
}

var socketPoolOnce sync.Once
var socketPoolInstance *socketPool

func GetSocketPool() *socketPool {
	socketPoolOnce.Do(func() {
		socketPoolInstance = new(socketPool)
	})
	return socketPoolInstance
}

func (s *socketPool) GetTCPSocket(addr string) Socket {
	return s.newTCPSocket(addr, false)
}

func (s *socketPool) GetSecuritySocket(addr string) Socket {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.socketMap == nil {
		s.socketMap = make(map[string]*list.List)
	}
	if _, ok := s.socketMap[addr]; !ok {
		s.socketMap[addr] = list.New()
	}
	socektList := s.socketMap[addr]
	if socektList.Len() == 0 {
		return s.newTCPSocket(addr, true)
	}
	sock := socektList.Front().Value.(Socket)
	socektList.Remove(socektList.Front())
	return sock
}

func (s *socketPool) Put(sock Socket) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := sock.(*socket.SecurityTCPSocket); ok {
		if s.socketMap == nil {
			s.socketMap = make(map[string]*list.List)
		}
		addr := sock.Addr()
		if _, ok := s.socketMap[addr]; !ok {
			s.socketMap[addr] = list.New()
		}
		socektList := s.socketMap[addr]
		if socektList.Len() > kMaxSocketPoolCount {
			fmt.Println("socketPool about kMaxSocketPoolCount")
			return
		}
		socektList.PushBack(sock)
	}
}

func (s *socketPool) newTCPSocket(addr string, isSecuritySocket bool) Socket {
	if isSecuritySocket {
		return NewSecurityTCPSocket(addr)
	} else {
		return NewTCPSocket(addr)
	}
}
