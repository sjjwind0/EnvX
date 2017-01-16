package main

import (
	"bufio"
	"conn"
	"conn/socket"
	"fmt"
	"io"
	"os"
	"proxy"
	"rule"
	"strconv"
	"strings"
)

var socketMap map[int]socket.Socket = make(map[int]socket.Socket)
var currentSid = 0

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

func handleSocket(sid int, sock socket.Socket) {
	var readBuffer []byte = make([]byte, 1024)
	for {
		size, err := sock.Read(readBuffer)
		if err == io.EOF {
			fmt.Println("continue")
			continue
		}
		fmt.Println("sid:", sid, "\trecv content:", string(readBuffer[:size]))
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
		startClient()
	}
	if isServer {
		startServer()
	}
}

func startClient() {
	fmt.Println("start client...")
	clientSocket, err := conn.Dial("sts", "127.0.0.1:12345")
	if err != nil {
		fmt.Println("dial failed:", err)
		return
	}
	fmt.Println("connect success")
	socketMap[currentSid] = clientSocket
	currentSid = currentSid + 1
	go handleSocket(currentSid-1, clientSocket)
}

func startServer() {
	fmt.Println("start server...")
	var listener socket.Listener
	var err error
	listener, err = conn.Listen("sts", "127.0.0.1:12345")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	go func() {
		defer listener.Close()
		for {
			acceptSocket, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting: ", err)
				return
			}
			socketMap[currentSid] = acceptSocket
			currentSid = currentSid + 1
			fmt.Println("accept success")
			go handleSocket(currentSid-1, acceptSocket)
		}
	}()
}

func sendMessage(sid int, msg string) {
	fmt.Println("send message: ", sid, msg)
	if sock, ok := socketMap[sid]; ok {
		_, err := sock.Write([]byte(msg))
		if err != nil {
			fmt.Println("send error:", err)
		}
	}
}

func listAllSocket() {
	fmt.Println("list all socket")
	for sid, _ := range socketMap {
		fmt.Println("socket:", sid)
	}
}

func closeSocket(sid int) {
	if sock, ok := socketMap[sid]; ok {
		sock.Close()
	}
}

func startMultiSocketTest() {
	for {
		fmt.Println("waiting command...")
		reader := bufio.NewReader(os.Stdin)
		command, _ := reader.ReadString('\n')
		fmt.Println("command:", command)
		switch command[:len(command)-1] {
		case "listen":
			startServer()
		case "list":
			listAllSocket()
		case "dial":
			startClient()
		case "stop":
		default:
			commandList := strings.Split(command, " ")
			if len(commandList) == 3 {
				switch commandList[0] {
				case "send":
					sid, _ := strconv.Atoi(commandList[1])
					msg := commandList[2]
					sendMessage(sid, msg)
				}
			} else if len(commandList) == 2 {
				switch commandList[0] {
				case "close":
					sid, _ := strconv.Atoi(commandList[1])
					closeSocket(sid)
				}
			}
		}
	}
}

func main() {
	startMultiSocketTest()
}
