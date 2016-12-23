package main

import (
	// "fmt"
	"proxy"
	"server"
	"time"
)

func main() {
	go func() {
		// waiting for client request
		server.StartServer("127.0.0.1", "8000")
	}()
	time.Sleep(time.Millisecond * 100)
	// remote proxy server addr
	p := proxy.NewProxy("127.0.0.1", "8000")
	// local proxy server addr
	p.StartProxyListener("127.0.0.1", "9000")
}
