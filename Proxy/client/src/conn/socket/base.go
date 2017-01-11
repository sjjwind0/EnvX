package socket

type Socket interface {
	Addr() string
	Accept() (Socket, error)
	Connect() error
	Read(readData []byte) (int, error)
	Write(writeData []byte) (int, error)
	Close()
}
