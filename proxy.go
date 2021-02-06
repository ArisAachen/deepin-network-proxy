package main

import (
	"net"

	com "github.com/DeepinProxy/Com"
	cfg "github.com/DeepinProxy/Config"
	tProxy "github.com/DeepinProxy/TProxy"
	"pkg.deepin.io/lib/log"
)

var logger = log.NewLogger("daemon/proxy")
var mgr *tProxy.HandlerMgr

// proxy main
func main() {
	var udp bool = true
	var lsp string = ":8080"
	var proto string = "sock5"

	mgr = tProxy.NewHandlerMsg()

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
		logger.Warning("convert udp data failed")
		return
	}
	err = com.SetConnOptTrn(conn)
	if err != nil {
		logger.Warningf("set conn opt transparent failed, err: %v", err)
		return
	}

	for {
		buf := make([]byte, 512)
		oob := make([]byte, 1024)
		_, oobNum, _, lAddr, err := conn.ReadMsgUDP(buf, oob)
		if err != nil {
			logger.Fatal(err)
		}

		// get real remote addr
		rBaseAddr, err := com.ParseRemoteAddrFromMsgHdr(oob[:oobNum])
		if err != nil {
			logger.Fatal(err)
		}

		// make remote addr
		rAddr := &net.UDPAddr{
			IP:   rBaseAddr.IP,
			Port: rBaseAddr.Port,
		}

		// func ProxyUdp(scope tProxy.ProxyScope, proxy cfg.Proxy, local net.Addr, remote net.Addr)
		go ProxyUdp(tProxy.GlobalProxy, proxy, lAddr, rAddr, buf)
	}
}

func ProxyUdp(scope tProxy.ProxyScope, proxy cfg.Proxy, lAddr net.Addr, rAddr net.Addr, buf []byte) {
	// make a fake udp dial to cheat socket
	lConn, err := com.MegaDial("udp", rAddr, lAddr)
	if err != nil {
		logger.Warningf("fake dial udp rAddr to lAddr failed, err: %v", err)
		return
	}
	// make key to mark this connection
	key := tProxy.HandlerKey{
		SrcAddr: lAddr.String(),
		DstAddr: rAddr.String(),
	}
	// create new handler
	handler := tProxy.NewHandler("sock5udp", scope, key, proxy, lAddr, rAddr, lConn)
	// create tunnel between proxy server and dst server
	err = handler.Tunnel()
	if err != nil {
		handler.Close()
		return
	}
	// add handler to map
	handler.AddMgr(mgr)
	// begin communication
	handler.Communicate()
	// write first buf to rAddr
	pkgData := com.DataPackage{
		Addr: rAddr,
		Data: buf,
	}
	// write first udp to remote
	err = handler.WriteRemote(com.MarshalPackage(pkgData, "udp"))
	if err != nil {
		handler.Close()
		return
	}
}

//func NewTcpProxy(listen string) {
//	l, err := net.Listen("tcp", listen)
//	if err != nil {
//		logger.Warningf("listen tcp port failed, err: %v", err)
//		return
//	}
//	defer l.Close()
//
//	config := cfg.NewProxyCfg()
//	err = config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
//	if err != nil {
//		logger.Warningf("load config failed, err: %v", err)
//		return
//	}
//
//	proxy, err := config.GetProxy("global", "sock5")
//	if err != nil {
//		logger.Warningf("get proxy from config failed, err: %v", err)
//		return
//	}
//
//	for {
//		localConn, err := l.Accept()
//		if err != nil {
//			logger.Warningf("accept socket failed, err: %v", err)
//			continue
//		}
//
//		// get remote addr
//		tcpCon, ok := localConn.(*net.TCPConn)
//		if !ok {
//			logger.Warningf("accept conn type is not tcp conn %v", err)
//			err = localConn.Close()
//			continue
//		}
//		tcpAddr, err := com.GetTcpRemoteAddr(tcpCon)
//		if err != nil {
//			logger.Warningf("get remote addr failed, err: %v", err)
//			err = localConn.Close()
//			continue
//		}
//		// create handler
//		handler := tProxy.NewHandler(localConn, proxy, "sock5")
//		// dial proxy server
//		addr := proxy.Server
//		port := strconv.Itoa(proxy.Port)
//		if port == "" {
//			port = "80"
//		}
//		remoteConn, err := net.DialTimeout("tcp", addr+":"+port, 3*time.Second)
//		if err != nil {
//			logger.Warningf("dial remote proxy server failed, err: %v", err)
//			err = localConn.Close()
//			continue
//		}
//		err = handler.Tunnel(remoteConn, tcpAddr)
//		if err != nil {
//			logger.Warningf("create tunnel failed, %v", err)
//			err = localConn.Close()
//			err = remoteConn.Close()
//		}
//		tProxy.Communicate(localConn, remoteConn)
//	}
//}
//
//func Handler() {
//
//}
