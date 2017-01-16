package socket

type Socket interface {
	Read(readData []byte) (int, error)
	Write(writeData []byte) (int, error)
	Close()
}
