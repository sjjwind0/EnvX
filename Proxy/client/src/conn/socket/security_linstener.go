package socket

type SecurityListener interface {
	OnClose(sock *VirtualSecurityTCPSocket)
}
