package main

import (
	"net"

	com "github.com/DeepinProxy/Com"
	cfg "github.com/DeepinProxy/Config"
	tProxy "github.com/DeepinProxy/TProxy"
)

// https://www.kernel.org/doc/Documentation/networking/tproxy.txt

// tcp proxy module

func NewTcpProxy(lsp string, proxyTyp tProxy.ProxyTyp, proxy cfg.Proxy) {
	// listen port
	l, err := net.Listen("tcp", lsp)
	if err != nil {
		logger.Warningf("[%s] listen port failed, err: %v", proxyTyp, err)
		return
	}
	defer l.Close()
	// convert to tcp listener
	tl, ok := l.(*net.TCPListener)
	if !ok {
		logger.Warningf("[%s] listener is not tcp listener type", proxyTyp)
		return
	}
	// get file
	file, err := tl.File()
	if err != nil {
		logger.Warningf("[%s] tcp listener get file failed, err: %v", err)
		return
	}
	// set transparent
	err = com.SetSockOptTrn(int(file.Fd()))
	if err != nil {
		logger.Warningf("[%s] set fd opt transparent failed, err: %v", proxyTyp, err)
		return
	}

	for {
		// accept connect
		lConn, err := l.Accept()
		if err != nil {
			logger.Warningf("[%s] accept socket failed, err: %v", proxyTyp, err)
		}

		go ProxyTcp(proxyTyp, tProxy.GlobalProxy, proxy, lConn)
	}
}

func ProxyTcp(proxyTyp tProxy.ProxyTyp, scope tProxy.ProxyScope, proxy cfg.Proxy, lConn net.Conn) {
	// request is redirect by t-proxy, output -> pre-routing
	// at that time, the actual remote addr is conn`s local addr, the actual local addr is conn`s remote addr
	// can use conn as fake remote conn, to connect with actual local connection
	lAddr := lConn.RemoteAddr()
	rAddr := lConn.LocalAddr()

	// print local -> remote
	logger.Debugf("[%s] tcp request capture by proxy successfully, "+
		"local[%s] -> remote [%s]", proxyTyp, lAddr.String(), rAddr.String())

	// make key to mark this connection
	key := tProxy.HandlerKey{
		SrcAddr: lAddr.String(),
		DstAddr: rAddr.String(),
	}
	// create new handler
	handler := tProxy.NewHandler(proxyTyp, tProxy.GlobalProxy, key, proxy, lAddr, rAddr, lConn)
	// create tunnel between proxy server and dst server
	err := handler.Tunnel()
	if err != nil {
		logger.Warningf("[%s] create tunnel failed, err: %v", proxyTyp, err)
		handler.Close()
		return
	}
	// add handler to map
	handler.AddMgr(mgr)
	// begin communication
	handler.Communicate()
}

// udp proxy module

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
		// read origin addr
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

// udp proxy core
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
	handler := tProxy.NewHandler(tProxy.SOCK5UDP, scope, key, proxy, lAddr, rAddr, lConn)
	// create tunnel between proxy server and dst server
	err = handler.Tunnel()
	if err != nil {
		logger.Warningf("[%s] create tunnel failed, err: %v", tProxy.SOCK5UDP, err)
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
