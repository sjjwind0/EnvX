package main

import (
	"fmt"
	"os"
	"proxy"
	"rule"
	"strings"
)

const (
	kStatMode_Unknown = 0x0
	kStartMode_Client = 0x1
	kStartMode_Server = 0x2
)

func main() {
	// start remote server
	var startMode int = kStatMode_Unknown
	var clientAddr string = ""
	var proxyAddr string = ""
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--client":
			startMode |= kStartMode_Client
		case "--server":
			startMode |= kStartMode_Server
		case "--help":
			fmt.Println("proxy [mode] addr\n--client client mode\n--server server mode\n--client-addr client listen addr\n--proxy-addr proxy listen addr")
		default:
			if strings.HasPrefix(arg, "--client-addr=") {
				clientAddr = arg[len("--client-addr="):]
			}
			if strings.HasPrefix(arg, "--proxy-addr=") {
				proxyAddr = arg[len("--proxy-addr="):]
			}
		}
	}

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
		proxy.NewNativeProxy(clientAddr, proxyAddr).StartListener()
	}
}
