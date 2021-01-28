package main

import (
	"net"
	"pkg.deepin.io/lib/log"
	"strconv"
	"time"

	com "github.com/DeepinProxy/Com"
	cfg "github.com/DeepinProxy/Config"
	pro "github.com/DeepinProxy/HttpProxy"
)

var logger = log.NewLogger("daemon/proxy")

func main() {
	NewUdpProxy()
}

func NewUdpProxy() {
	l, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		logger.Warningf("listen udp port failed, err: %v", err)
		return
	}
	defer l.Close()



}

func NewTcpProxy(listen string) {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		logger.Warningf("listen tcp port failed, err: %v", err)
		return
	}
	defer l.Close()

	config := cfg.NewProxyCfg()
	err = config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
	if err != nil {
		logger.Warningf("load config failed, err: %v", err)
		return
	}

	proxy, err := config.GetProxy("global", "sock5")
	if err != nil {
		logger.Warningf("get proxy from config failed, err: %v", err)
		return
	}

	for {
		localConn, err := l.Accept()
		if err != nil {
			logger.Warningf("accept socket failed, err: %v", err)
			continue
		}

		// get remote addr
		tcpCon, ok := localConn.(*net.TCPConn)
		if !ok {
			logger.Warningf("accept conn type is not tcp conn %v", err)
			err = localConn.Close()
			continue
		}
		tcpAddr, err := com.GetTcpRemoteAddr(tcpCon)
		if err != nil {
			logger.Warningf("get remote addr failed, err: %v", err)
			err = localConn.Close()
			continue
		}
		// create handler
		handler := pro.NewHandler(localConn, proxy, "sock5")
		// dial proxy server
		addr := proxy.Server
		port := strconv.Itoa(proxy.Port)
		if port == "" {
			port = "80"
		}
		remoteConn, err := net.DialTimeout("tcp", addr+":"+port, 3*time.Second)
		if err != nil {
			logger.Warningf("dial remote proxy server failed, err: %v", err)
			err = localConn.Close()
			continue
		}
		err = handler.Tunnel(remoteConn, tcpAddr)
		if err != nil {
			logger.Warningf("create tunnel failed, %v", err)
			err = localConn.Close()
			err = remoteConn.Close()
		}
		pro.Communicate(localConn, remoteConn)
	}
}
