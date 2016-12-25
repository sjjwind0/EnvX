package conn

import (
	"fmt"
	"io"
	"net"
)

const kCacheBuffer = 4 * 1024

type Conn struct {
	c net.Conn
}

func CopyConn(conn net.Conn) *Conn {
	return &Conn{c: conn}
}

func Copy(src io.Writer, dst io.Reader) error {
	var readBuffer []byte = make([]byte, kCacheBuffer)
	for {
		readSize, err := dst.Read(readBuffer)
		if err != nil && err != io.EOF {
			return err
		}
		_, writeErr := src.Write(readBuffer[:readSize])
		if writeErr != nil {
			return writeErr
		}
		if err == io.EOF {
			return nil
		}
	}
}

func CopyToNative(src io.Writer, dst io.Reader) error {
	return nil
}

func NewTCPConn(addr string) (*Conn, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Conn dial", addr, "failed:", err)
		return nil, err
	}
	return &Conn{c}, err
}

func (c *Conn) Write(data []byte) (int, error) {
	var totalWriteSize int = 0
	var dataSize int = len(data)
	for {
		writeSize, err := c.c.Write(data)
		if err != nil && err != io.EOF {
			fmt.Println("Conn write failed:", err)
			return totalWriteSize, err
		}
		totalWriteSize += writeSize
		if totalWriteSize == dataSize {
			break
		}
		if err == io.EOF {
			break
		}
	}
	return totalWriteSize, nil
}

func (c *Conn) Read(data []byte) (int, error) {
	return c.c.Read(data)
}

func (c *Conn) ReadWithDecrypt() {

}

func (c *Conn) Ping() {
	// TODO: impl ping
}
