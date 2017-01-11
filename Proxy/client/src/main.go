package main

import (
	"conn"
	"conn/socket"
	"fmt"
	"io"
	"os"
	"proxy"
	"rule"
	"strings"
	"time"
)

const (
	kStatMode_Unknown = 0x0
	kStartMode_Client = 0x1
	kStartMode_Server = 0x2
)

func startProxyTest() {
	// start remote server
	var startMode int = kStatMode_Unknown
	var clientAddr string = ""
	var proxyAddr string = ""
	var securityMode bool = false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--client":
			startMode |= kStartMode_Client
		case "--server":
			startMode |= kStartMode_Server
		case "--help":
			fmt.Println("proxy [mode] addr\n--client client mode\n--server server mode\n--client-addr client listen addr\n--proxy-addr proxy listen addr")
		case "--security":
			securityMode = true
		default:
			if strings.HasPrefix(arg, "--client-addr=") {
				clientAddr = arg[len("--client-addr="):]
			}
			if strings.HasPrefix(arg, "--proxy-addr=") {
				proxyAddr = arg[len("--proxy-addr="):]
			}
		}
	}

	fmt.Println("mode: ", securityMode)

	fmt.Println("client-addr: ", clientAddr)
	fmt.Println("proxy-addr: ", proxyAddr)

	if startMode&kStartMode_Server != 0 {
		if startMode&kStartMode_Client != 0 {
			go proxy.NewRemoteProxy(proxyAddr).StartListener()
		} else {
			proxy.NewRemoteProxy(proxyAddr).StartListener()
		}
	}

	if startMode&kStartMode_Client != 0 {
		var c chan bool = make(chan bool)
		fmt.Println("download GFWList...")
		r := rule.NewGFWRuleParser()
		r.DownloadGFWRuleListFromIntenet(func(err error) {
			if err != nil {
				fmt.Println("download failed:", err)
				return
			}
			fmt.Println("download success")
			r.ParseFile()
			c <- true
		})
		<-c
		fmt.Println("start client")
		proxy.NewNativeProxy(clientAddr, proxyAddr).StartListener()
	}
}

func handleSocket(sock socket.Socket) {
	go func() {
		var readBuffer []byte = make([]byte, 1024)
		for {
			size, err := sock.Read(readBuffer)
			if err == io.EOF {
				time.Sleep(time.Millisecond * 500)
				continue
			}
			fmt.Println(string(readBuffer[:size]))
		}
	}()
	for {
		var readString string
		fmt.Scanln(&readString)
		fmt.Println("read string:", readString)
		switch readString {
		case "quit":
			os.Exit(0)
		case "close":
			sock.Close()
		default:
			sock.Write([]byte(readString))
		}
	}
}

func startSocketTest() {
	var isClient bool = false
	var isServer bool = false
	for _, args := range os.Args[1:] {
		switch args {
		case "--client":
			isClient = true
		case "--server":
			isServer = true
		}
	}
	if isClient {
		clientSocket, err := conn.Dial("sts", "127.0.0.1:12345")
		if err != nil {
			fmt.Println("dial failed:", err)
			return
		}
		fmt.Println("connect success")
		handleSocket(clientSocket)
	}
	if isServer {
		var listener socket.Listener
		var err error
		listener, err = conn.Listen("sts", "127.0.0.1:12345")
		if err != nil {
			fmt.Println("Error listening:", err)
			return
		}
		defer listener.Close()
		for {
			acceptSocket, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting: ", err)
				return
			}
			go func(acceptSocket socket.Socket) {
				defer acceptSocket.Close()
				handleSocket(acceptSocket)
			}(acceptSocket)
		}
	}
}

func main() {
	startSocketTest()
}
