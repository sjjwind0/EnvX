package socket

import (
	"fmt"
	"net"
	"sync"
)

type SecurityListener interface {
	OnClose(sock *VirtualSecurityTCPSocket)
}

func SecurityTCPSocketListen(addr string) (*SecurityTCPListener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	sock := NewSecurityTCPListener(addr, listener)
	sock.startBackgroundAcceptTask()
	return sock, err
}

type SecurityTCPListener struct {
	addr     string
	listener net.Listener

	acceptSocket            chan Socket
	acceptError             chan error
	allAcceptRealSocketList []*RealSecurityTCPSocket
	close                   bool
}

func NewSecurityTCPListener(addr string, netListener net.Listener) *SecurityTCPListener {
	return &SecurityTCPListener{addr: addr, listener: netListener}
}

func (s *SecurityTCPListener) Addr() string {
	return s.addr
}

func (s *SecurityTCPListener) startBackgroundAcceptTask() {
	s.acceptSocket = make(chan Socket)
	s.acceptError = make(chan error)
	go func() {
		for {
			tcpSocket, err := s.listener.Accept()
			if err != nil || s.close {
				fmt.Println("RealSecurityTCPSocket Accept failed:", err)
				s.acceptSocket <- nil
				s.acceptError <- err
				return
			}
			realSecurityTCPSocket := NewRealSecurityTCPSocket(tcpSocket)
			fmt.Println("real socket accept aravial")
			err = realSecurityTCPSocket.waitingConnect()
			if err != nil {
				s.acceptSocket <- nil
				s.acceptError <- err
				return
			}
			realSecurityTCPSocket.startBackgroundReadTask()
			realSecurityTCPSocket.startBackgroundWriteTask()

			s.allAcceptRealSocketList = append(s.allAcceptRealSocketList, realSecurityTCPSocket)
			go func() {
				if realSecurityTCPSocket.acceptLocker == nil {
					realSecurityTCPSocket.acceptLocker = new(sync.Mutex)
					realSecurityTCPSocket.acceptCond = sync.NewCond(realSecurityTCPSocket.acceptLocker)
				}
				for {
					realSecurityTCPSocket.acceptCond.L.Lock()
					if s.close {
						realSecurityTCPSocket.acceptCond.L.Unlock()
						s.acceptSocket <- nil
						s.acceptError <- nil
						return
					}
					realSecurityTCPSocket.acceptCond.Wait()

					s.acceptSocket <- realSecurityTCPSocket.NewVirtualSocket()
					s.acceptError <- nil
					realSecurityTCPSocket.acceptCond.L.Unlock()
				}
			}()

			s.acceptSocket <- realSecurityTCPSocket.NewVirtualSocket()
			s.acceptError <- err
		}
	}()
}

func (s *SecurityTCPListener) Accept() (Socket, error) {
	var newSocket Socket = <-s.acceptSocket
	var err error = <-s.acceptError
	fmt.Println("socket:", newSocket)
	fmt.Println("error:", err)
	return newSocket, err
}

func (s *SecurityTCPListener) Close() {
	s.close = true
	s.listener.Close()
	for _, sock := range s.allAcceptRealSocketList {
		sock.acceptCond.Signal()
	}
}
