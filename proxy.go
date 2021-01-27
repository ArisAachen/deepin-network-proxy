package main

import (
	cfg "github.com/proxyTest/Config"
	pro "github.com/proxyTest/HttpProxy"
	"log"
	"net"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()

	config := cfg.NewProxyCfg()
	err = config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
	if err != nil {
		log.Fatal(err)
	}

	proxy, err := config.GetProxy("global", "http")
	if err != nil {
		log.Fatal(err)
	}

	for {
		CConn, err := l.Accept()
		if err != nil {
			log.Println("accept is ", err)
			return
		}

		handler := pro.NewSock5Handler(CConn, proxy)
		if handler == nil {
			continue
		}
		handler.Communicate()
	}
}
