package socket

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func startListener(addr string, newConnCallback func(socket *SecurityTCPSocket)) {
	var listener net.Listener
	var err error
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	for {
		newConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			return
		}
		go func(netConn net.Conn) {
			defer netConn.Close()
			fmt.Println("new socket")
			socket := NewSecurityServerTCPSocket(netConn)
			newConnCallback(socket)
		}(newConn)
	}
}

func TestSocket(t *testing.T) {
	s := make(chan bool)
	s1 := make(chan bool)
	go startListener("127.0.0.1:10086", func(socket *SecurityTCPSocket) {
		defer func() {
			s <- true
		}()
		err := socket.WaitingConnect()
		if err != nil {
			fmt.Println("waiting error: ", err)
			t.Error(err)
		}
		var data []byte = make([]byte, 1024)
		readSize, err := socket.Read(data)
		if err != nil && err != io.EOF {
			t.Error(err)
		}
		fmt.Println("read: ", readSize)
		fmt.Println("data: ", string(data[:readSize])+"#")
		fmt.Println("real: ", "Hello World#")
		if string(data[:readSize]) != "Hello World" {
			t.Error("error data")
		}

		_, err = socket.Write([]byte("哼哼哈嘿，煎饼果子来一套"))
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < 100; i++ {
			_, err := socket.Write([]byte("哼哼哈嘿，煎饼果子来一套; "))
			fmt.Println("err: ", err)
		}

		socket.Close()
	})
	go func() {
		defer func() {
			s1 <- true
		}()
		socket := NewSecurityClientTCPSocket("127.0.0.1:10086")
		err := socket.Connect()
		if err != nil && err != io.EOF {
			t.Error(err)
			return
		}
		fmt.Println("connect ok")
		writeSize, err := socket.Write([]byte("Hello World"))
		if err != nil {
			t.Error(err)
		}
		fmt.Println("writeSize: ", writeSize)
		var testReadData1 []byte = make([]byte, 1024)

		time.Sleep(time.Second * 1)

		for i := 0; i < 5; i++ {

			testReadData1Size, err := socket.Read(testReadData1)
			t.Error("readSize: ", testReadData1Size, "\terr: ", err)
			if err != nil && err != io.EOF {
				t.Error("readSize: ", testReadData1Size, "\terr: ", err)
			}

			fmt.Println("lalal data: ", string(testReadData1[:testReadData1Size]))
			if string(testReadData1[:testReadData1Size]) != "哼哼哈嘿，煎饼果子来一套" {
				t.Error("error data")
			}
		}

		socket.Stop()

	}()
	<-s
	<-s1
}
