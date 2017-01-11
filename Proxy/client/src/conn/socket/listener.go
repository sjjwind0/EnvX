package socket

type Listener interface {
	Accept() (Socket, error)
	Close()
	Addr() string
}
