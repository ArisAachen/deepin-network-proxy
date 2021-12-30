package DBus

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	com "github.com/ArisAachen/deepin-network-proxy/com"
	config "github.com/ArisAachen/deepin-network-proxy/config"
	newCGroups "github.com/ArisAachen/deepin-network-proxy/new_cgroups"
	tProxy "github.com/ArisAachen/deepin-network-proxy/tproxy"
	"github.com/godbus/dbus"
	"github.com/linuxdeepin/go-lib/dbusutil"
)

// interface path
func (mgr *proxyPrv) GetInterfaceName() string {
	return BusInterface + "." + mgr.scope.String()
}

// get proxy
func (mgr *proxyPrv) GetProxy() (string, *dbus.Error) {
	if mgr.Proxy.ProtoType == "" {
		return "", nil
	}
	buf, err := com.MarshalJson(mgr.Proxy)
	if err != nil {
		logger.Warningf("[%s] get proxy failed, err: %v", mgr.scope, err)
		return "", dbusutil.ToError(err)
	}
	return buf, nil
}

// start proxy
func (mgr *proxyPrv) StartProxy(sender dbus.Sender, proto string, name string, udp bool) *dbus.Error {
	con, err := dbusutil.NewSystemService()
	if err != nil {
		logger.Warningf("get session service failed, err: %v", err)
		return dbusutil.ToError(err)
	}
	mgr.uid, err = con.GetConnUID(string(sender))
	if err != nil {
		logger.Warningf("get name owner failed, err: %v", err)
		return dbusutil.ToError(err)
	}
	id, err := user.LookupId(strconv.Itoa(int(mgr.uid)))
	if err != nil {
		return dbusutil.ToError(err)
	}
	gid, err := strconv.Atoi(id.Gid)
	if err != nil {
		return dbusutil.ToError(err)
	}
	mgr.gid = uint32(gid)
	if mgr.Enabled {
		_ = mgr.StopProxy()
	}

	//// already in proxy
	//if !mgr.stop {
	//	logger.Debugf("[%] already in proxy", mgr.scope)
	//	return nil
	//}
	//mgr.stop = false
	logger.Debugf("[%s] start proxy, proto [%s] name [%s] udp [%v]", mgr.scope, proto, name, udp)
	// check if proto is legal
	var proxyTyp tProxy.ProtoTyp
	if proto == "socks5" {
		// never err
		proxyTyp = tProxy.SOCK5TCP
	} else {
		proxyTyp, err = tProxy.BuildProto(proto)
		if err != nil {
			return dbusutil.ToError(err)
		}
	}
	// get proxies
	proxy, err := mgr.Proxies.GetProxy(proto, name)
	if err != nil {
		logger.Warningf("[%s] get proxy failed, err: %v", mgr.scope, err)
		return dbusutil.ToError(err)
	}
	// save proxy
	mgr.Proxy = proxy
	logger.Debugf("[%s] get proxy success, proxy: %v", mgr.scope, proxy)
	// tcp module
	listen, err := mgr.listen()
	if err != nil {
		return dbusutil.ToError(err)
	}
	// save tcp handler
	mgr.tcpHandler = listen
	logger.Debugf("[%s] proxy [%s] listen tcp success at port %v", mgr.scope, proto, mgr.Proxies.TPort)
	// in case blocks DBus-return, use goroutine
	go mgr.accept(proxyTyp, proxy, listen)

	// udp module
	if udp && proto == "sock5" {
		// listen packet conn
		packetConn, err := mgr.listenPacket()
		if err != nil {
			return dbusutil.ToError(err)
		}
		logger.Debugf("[%s] proxy [%s] listen udp success at port %v", mgr.scope, proto, mgr.Proxies.TPort)
		// start proxy udp
		go mgr.readMsgUDP(proxyTyp, proxy, packetConn)
	}

	// mark enable
	mgr.Enabled = true

	err = mgr.startRedirect()
	if err != nil {
		logger.Warningf("start redirect failed, err: %v", err)
		return dbusutil.ToError(err)
	}

	go func() {
		err := mgr.dnsProxy.startDNSProxy()
		if err != nil {
			logger.Warningf("start dns proxy failed: %v", err)
		}
	}()

	return nil
}

// stop proxy
func (mgr *proxyPrv) StopProxy() *dbus.Error {
	if !mgr.Enabled {
		return nil
	}
	//if mgr.stop {
	//	logger.Debugf("[%s] already stop proxy")
	//	return nil
	//}
	//mgr.stop = true
	logger.Debugf("[%s] stop proxy, enable: %v, proxy: %v", mgr.scope, mgr.Enabled, mgr.Proxy)
	// stop to break accept and read message
	if mgr.tcpHandler != nil {
		err := mgr.tcpHandler.Close()
		if err != nil {
			logger.Warningf("[%s] stop proxy tcp handler failed, err: %v", mgr.scope, err)
		}
	}
	if mgr.udpHandler != nil {
		err := mgr.udpHandler.Close()
		if err != nil {
			logger.Warningf("[%s] stop proxy udp handler failed, err: %v", mgr.scope, err)
		}
	}

	mgr.Enabled = false

	err := mgr.stopRedirect()
	if err != nil {
		logger.Warningf("stop redirect failed, err: %v", err)
		return dbusutil.ToError(err)
	}

	return nil
}

// attach to cgroup v2 user
func (mgr *proxyPrv) attachBackUser() error {
	pathSl := []string{"/sys", "fs", "cgroup", "unified", "user.slice", "user-" + strconv.Itoa(int(mgr.uid)) + ".slice", "cgroup.procs"}
	path := strings.Join(pathSl, "/")
	logger.Debugf("attach back cgroup user is %s", path)
	ctl := mgr.controller.GetControlPath()
	if _, err := os.Stat(ctl); err != nil {
		logger.Warningf("attach back file not exist, err: %v", err)
		return err
	}
	// read cgroups file
	buf, err := ioutil.ReadFile(ctl)
	if err != nil {
		logger.Warningf("read file failed, err: %v", err)
		return err
	}
	// construct buffer
	by := bytes.NewBuffer(buf)
	// make reader
	rd := bufio.NewReader(by)
	for {
		buf, _, err = rd.ReadLine()
		if err != nil {
			logger.Debugf("read file end, reason: %v", err)
			return err
		}
		// attach back
		err = newCGroups.Attach(string(buf), path)
		if err != nil {
			logger.Debugf("attach pid %s back %s failed, err: %v", string(buf), path, err)
			continue
		}
		logger.Debugf("attach pid %s back %s success", string(buf), path)
	}

}

// set proxy
func (mgr *proxyPrv) AddProxy(proto string, name string, jsonProxy []byte) *dbus.Error {
	proxy, err := UnMarshalProxy(jsonProxy)
	if err != nil {
		logger.Warningf("[%s] unmarshal proxy message failed, err: %v", mgr.scope, err)
		return dbusutil.ToError(err)
	}
	// check if exist
	mgr.Proxies.SetProxy(proto, name, proxy)
	return nil
}

// set proxies
func (mgr *proxyPrv) SetProxies(proxies config.ScopeProxies) *dbus.Error {
	mgr.Proxies = proxies
	err := mgr.writeConfig()
	if err != nil {
		logger.Warningf("[%s] write config failed, err: %v", mgr.scope, err)
		return dbusutil.ToError(err)
	}
	return nil
}

func (mgr *proxyPrv) ClearProxy() *dbus.Error {
	mgr.Proxies.Proxies = nil
	err := mgr.writeConfig()
	if err != nil {
		logger.Warningf("[%s] write config failed, err: %v", mgr.scope, err)
		return dbusutil.ToError(err)
	}
	return nil
}

// set tcp opt listen
func (mgr *proxyPrv) listen() (net.Listener, error) {
	// get proxies
	tp := strconv.Itoa(mgr.Proxies.TPort)
	l, err := net.Listen("tcp", ":"+tp)
	if err != nil {
		logger.Warningf("[%s] listen port failed, err: %v", mgr.scope, err)
		return nil, err
	}
	// convert to tcp listener
	tl, ok := l.(*net.TCPListener)
	if !ok {
		logger.Warningf("[%s] listener is not tcp listener type", mgr.scope)
		return nil, err
	}
	// get file
	file, err := tl.File()
	if err != nil {
		logger.Warningf("[%s] tcp listener get file failed, err: %v", err)
		return nil, err
	}
	defer file.Close()
	// set transparent
	err = com.SetSockOptTrn(int(file.Fd()))
	if err != nil {
		logger.Warningf("[%s] set fd opt transparent failed, err: %v", mgr.scope, err)
		return nil, err
	}
	// set non block
	err = syscall.SetNonblock(int(file.Fd()), true)
	if err != nil {
		logger.Warningf("[%s] set non block failed, err: %v", mgr.scope, err)
		return nil, err
	}

	return l, nil
}

// set udp opt listen
func (mgr *proxyPrv) listenPacket() (net.PacketConn, error) {
	// get proxies
	tp := strconv.Itoa(mgr.Proxies.TPort)
	l, err := net.ListenPacket("udp", ":"+tp)
	if err != nil {
		logger.Warningf("[%s] listen udp package port failed, err: %v", mgr.scope, err)
		return nil, err
	}
	// ip_transparent
	conn, ok := l.(*net.UDPConn)
	if !ok {
		logger.Warning("convert udp data failed")
		return nil, err
	}
	err = com.SetConnOptTrn(conn)
	if err != nil {
		logger.Warningf("set conn opt transparent failed, err: %v", err)
		return nil, err
	}
	return l, nil
}

// proxy tcp
func (mgr *proxyPrv) accept(proxyTyp tProxy.ProtoTyp, proxy config.Proxy, listen net.Listener) {
	if listen == nil {
		logger.Warningf("[%s] tcp listener is nil", mgr.scope)
		return
	}

	// start accept until stop
	for {
		// accept connect
		// https://github.com/golang/go/issues/10527
		lConn, err := listen.Accept()
		if err != nil {
			if !mgr.Enabled {
				logger.Debugf("[%s] stop proxy tcp break", mgr.scope)
				break
			}
			logger.Warningf("[%s] accept socket failed, err: %v", proxyTyp, err)
			break
		}
		// proxy tcp
		go mgr.proxyTcp(proxyTyp, proxy, lConn)
	}
	logger.Debugf("[%s] stop proxy, prepare close handler", mgr.scope)
	mgr.handlerMgr.CloseTypHandler(proxyTyp)
	mgr.tcpHandler = nil
}

// read udp message
func (mgr *proxyPrv) readMsgUDP(proxyTyp tProxy.ProtoTyp, proxy config.Proxy, listen net.PacketConn) {
	if listen == nil {
		logger.Warningf("[%s] tcp listener is nil", mgr.scope)
		return
	}
	// ip_transparent
	conn, ok := listen.(*net.UDPConn)
	if !ok {
		logger.Warning("convert udp data failed")
		return
	}
	defer conn.Close()

	// start accept until stop
	for {
		// read origin addr
		buf := make([]byte, 512)
		oob := make([]byte, 1024)
		_, oobNum, _, lAddr, err := conn.ReadMsgUDP(buf, oob)
		if err != nil {
			if !mgr.Enabled {
				logger.Debugf("[%s] stop proxy udp break", mgr.scope)
				break
			}
			logger.Warningf("[%s] read udp msg failed, err: %v", mgr.scope, err)
			continue
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
		// proxy udp
		go mgr.proxyUdp(proxy, lAddr, rAddr, buf)
	}
	logger.Debugf("[%s] stop proxy, prepare close handler", mgr.scope)
	mgr.handlerMgr.CloseTypHandler(proxyTyp)
}

type DomainAddr struct {
	network string
	Domain  string
	Port    int
}

func newDomainAddr(network string, domain string, port int) *DomainAddr {
	return &DomainAddr{
		network: network,
		Domain:  domain,
		Port:    port,
	}
}

func (a *DomainAddr) Network() string {
	return a.network
}

func (a *DomainAddr) String() string {
	return a.Domain + ":" + strconv.Itoa(a.Port)
}

// for t-proxy
func (mgr *proxyPrv) proxyTcp(proxyTyp tProxy.ProtoTyp, proxy config.Proxy, lConn net.Conn) {
	// request is redirect by t-proxy, output -> pre-routing
	// at that time, the actual remote addr is conn`s local addr, the actual local addr is conn`s remote addr
	// can use conn as fake remote conn, to connect with actual local connection
	lAddr := lConn.RemoteAddr()
	rAddr := lConn.LocalAddr()

	realRAddr := rAddr
	if proxyTyp == tProxy.HTTP {
		switch addr := rAddr.(type) {
		case *net.UDPAddr:
			domain, ok := mgr.dnsProxy.getDomainFromFakeIP(addr.IP)
			if ok {
				realRAddr = newDomainAddr("udp", domain, addr.Port)
			}

		case *net.TCPAddr:
			domain, ok := mgr.dnsProxy.getDomainFromFakeIP(addr.IP)
			if ok {
				realRAddr = newDomainAddr("tcp", domain, addr.Port)
			}

		}
	}

	// print local -> remote
	logger.Infof("[%s] tcp request capture by proxy successfully, "+
		"local[%s] -> remote [%s](%s)", proxyTyp, lAddr.String(), rAddr.String(), realRAddr)

	// make key to mark this connection
	key := tProxy.HandlerKey{
		SrcAddr: lAddr.String(),
		DstAddr: rAddr.String(),
	}
	// create new handler
	handler := tProxy.NewHandler(proxyTyp, mgr.scope, key, proxy, lAddr, realRAddr, lConn)
	// create tunnel between proxy server and dst server
	err := handler.Tunnel()
	if err != nil {
		logger.Warningf("[%s] create tunnel failed, err: %v", proxyTyp, err)
		handler.Close()
		return
	}
	// add handler to map
	handler.AddMgr(mgr.handlerMgr)
	// begin communication
	handler.Communicate()
}

func (mgr *proxyPrv) proxyUdp(proxy config.Proxy, lAddr net.Addr, rAddr net.Addr, buf []byte) {
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
	handler := tProxy.NewHandler(tProxy.SOCK5UDP, mgr.scope, key, proxy, lAddr, rAddr, lConn)
	// create tunnel between proxy server and dst server
	err = handler.Tunnel()
	if err != nil {
		logger.Warningf("[%s] create tunnel failed, err: %v", tProxy.SOCK5UDP, err)
		handler.Close()
		return
	}
	// add handler to map
	handler.AddMgr(mgr.handlerMgr)
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
