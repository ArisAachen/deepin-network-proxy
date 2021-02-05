package main

import (
	"net"
	"pkg.deepin.io/lib/log"
	"strconv"
	"time"

	com "github.com/DeepinProxy/Com"
	cfg "github.com/DeepinProxy/Config"
	tcpProxy "github.com/DeepinProxy/TcpProxy"
	udpProxy "github.com/DeepinProxy/UdpProxy"
)

var logger = log.NewLogger("daemon/proxy")

// proxy main
func main() {
	var udp bool = true
	var lsp string = ":8080"
	var proto string = "sock5"

	// get config
	config := cfg.NewProxyCfg()
	err := config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
	if err != nil {
		logger.Warningf("load config failed, err: %v", err)
		return
	}

	proxy, err := config.GetProxy("global", proto)
	if err != nil {
		logger.Warningf("get proxy from config failed, err: %v", err)
		return
	}

	if udp && proto == "sock5" {
		NewUdpProxy(lsp, proxy)
	}
}

// https://www.kernel.org/doc/Documentation/networking/tproxy.txt
func NewUdpProxy(lsp string, proxy cfg.Proxy) {
	l, err := net.ListenPacket("udp", lsp)
	if err != nil {
		logger.Warningf("listen udp package port failed, err: %v", err)
		return
	}
	defer l.Close()

	// ip_transparent
	conn, ok := l.(*net.UDPConn)
	if !ok {
		logger.Warning("")
	}
	err = com.SetConnOptTrn(conn)

	addr := proxy.Server
	port := strconv.Itoa(proxy.Port)
	if port == "" {
		port = "80"
	}

	for {
		buf := make([]byte, 512)
		oob := make([]byte, 1024)
		_, oobNum, _, lAddr, err := conn.ReadMsgUDP(buf, oob)
		if err != nil {
			logger.Fatal(err)
		}

		// get real remote addr
		rAddr, err := com.ParseRemoteAddrFromMsgHdr(oob[:oobNum])
		if err != nil {
			logger.Fatal(err)
		}

		// make a fake udp dial to cheat socket
		lConn, err := com.MegaDial("udp", rAddr, lAddr)
		if err != nil {
			logger.Warningf("fake dial udp remote to local failed, err: %v", err)
		}

		logger.Infof("recv buf is %v", buf)

		remoteConn, err := net.DialTimeout("tcp", addr+":"+port, 3*time.Second)
		if err != nil {
			logger.Warning(err)
			continue
		}

		handler := udpProxy.NewSock5Handler(lConn, proxy)
		err = handler.Tunnel(remoteConn, rAddr)
		if err != nil {
			logger.Warningf("tunnel failed, err: %v", err)
			continue
		}

		buf = udpProxy.MarshalUdpPackage(com.DataPackage{Addr: rAddr, Data: buf})
		_, err = handler.RemoteHandler.Write(buf)
		if err != nil {
			logger.Fatal(err)
		}

		buf = make([]byte, 512)
		_, err = handler.RemoteHandler.Read(buf)
		if err != nil {
			logger.Fatal(err)
		}
		pkg := com.UnMarshalPackage(buf)

		//pConn, err := udpProxy.MegaDial(rAddr, lAddr)
		//if err != nil {
		//	logger.Fatalf("dial failed, err: %v", err)
		//}
		//
		//_, err = pConn.Write(pkg.Data)
		//if err != nil {
		//	logger.Fatal("dial failed, err: %v", err)
		//}
		//buf = make([]byte, 512)
		//_, err = pConn.Read(buf)
		//if err != nil {
		//	logger.Fatal("dial failed, err: %v", err)
		//}
	}
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
		handler := tcpProxy.NewHandler(localConn, proxy, "sock5")
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
		tcpProxy.Communicate(localConn, remoteConn)
	}
}
