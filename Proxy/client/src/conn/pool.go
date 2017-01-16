package conn

import (
	"conn/socket"
	"container/list"
	"fmt"
	"sync"
	"util"
)

const (
	kMaxSocketPoolCount = 20
)

type socketPool struct {
	socketMap       map[string]*list.List
	mutex           sync.Mutex
	backgroundTimer *util.Timer
}

var socketPoolOnce sync.Once
var socketPoolInstance *socketPool

func GetSocketPool() *socketPool {
	socketPoolOnce.Do(func() {
		socketPoolInstance = new(socketPool)
		socketPoolInstance.StartBackgroundHandler()
	})
	return socketPoolInstance
}

func (s *socketPool) GetTCPSocket(addr string) socket.Socket {
	return s.newTCPSocket(addr, false)
}

func (s *socketPool) GetSecuritySocket(addr string) socket.Socket {
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
	sock := socektList.Front().Value.(socket.Socket)
	socektList.Remove(socektList.Front())
	return sock
}

func (s *socketPool) Put(sock socket.Socket) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	//if _, ok := sock.(*socket.VirtualSecurityTCPSocket); ok {
	//	if s.socketMap == nil {
	//		s.socketMap = make(map[string]*list.List)
	//	}
	//	addr := sock.Addr()
	//	if _, ok := s.socketMap[addr]; !ok {
	//		s.socketMap[addr] = list.New()
	//	}
	//	socektList := s.socketMap[addr]
	//	if socektList.Len() > kMaxSocketPoolCount {
	//		fmt.Println("socketPool about kMaxSocketPoolCount")
	//		return
	//	}
	//	socektList.PushBack(sock)
	//}
}

func (s *socketPool) Remove(sock socket.Socket) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	//if _, ok := sock.(*socket.VirtualSecurityTCPSocket); ok {
	//	if s.socketMap == nil {
	//		return
	//	}
	//	addr := sock.Addr()
	//	if _, ok := s.socketMap[addr]; !ok {
	//		return
	//	}
	//	socektList := s.socketMap[addr]
	//	for iter := socektList.Front(); iter != nil; iter = iter.Next() {
	//		if iter.Value.(socket.Socket) == sock {
	//			socektList.Remove(iter)
	//		}
	//	}
	//}
}

func (s *socketPool) newTCPSocket(addr string, isSecuritySocket bool) socket.Socket {
	// if isSecuritySocket {
	// 	return NewSecurityTCPSocket(addr)
	// } else {
	// 	return NewTCPSocket(addr)
	// }
	return nil
}

func (s *socketPool) StartBackgroundHandler() {
	// send ping to server every 10 minutes.
	//s.backgroundTimer = util.NewRepeatingTimer()
	//s.backgroundTimer.Start(time.Minute*10, func() {
	//	s.mutex.Lock()
	//	defer s.mutex.Unlock()
	//
	//	for _, socketList := range s.socketMap {
	//		for iter := socketList.Front(); iter != nil; iter = iter.Next() {
	//			sock := iter.Value.(*socket.VirtualSecurityTCPSocket)
	//			sock.Ping()
	//		}
	//	}
	//})
}

func (s *socketPool) StopBackgroundHandler() {
	s.backgroundTimer.Stop()
}

func (s *socketPool) OnClose(sock *socket.VirtualSecurityTCPSocket) {
	fmt.Println("on close")
	s.Put(sock)
}
