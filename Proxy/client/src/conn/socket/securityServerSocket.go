package socket

type securityServerSocket struct {
	currentSid      int
	allAcceptSocket map[int]*SecurityTCPSocket
}

func (s *securityServerSocket) Accept() (Socket, error) {
	acceptSid := currentSid
	s.currentSid = s.currentSid + 1
	acceptSocket := newSecurityServerTCPSocket(acceptSid, s)
	s.allAcceptSocket[acceptSid] = acceptSocket
	acceptSocket.listenTCPSocket = s
	return acceptSocket, nil
}

func (s *securityServerSocket) Addr() string {
	return ""
}

func (s *securityServerSocket) Close() {
}
