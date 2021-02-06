package TProxy

import (
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/DeepinProxy/Config"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

// handler module

type BaseHandler interface {
	Tunnel() error
	Communicate()
	Close()
}

type ProxyScope int

const (
	GlobalProxy ProxyScope = iota
	AppProxy
)

// proxy server
type proxyServer struct {
	server string
	port   int
	auth
}

// auth message
type auth struct {
	user     string
	password string
}

// handler key in case keep the same handler
type HandlerKey struct {
	SrcAddr string
	DstAddr string
}

// manager all handler
type HandlerMgr struct {
	handlerLock sync.RWMutex
	handlerMap  map[ProxyScope]map[HandlerKey]BaseHandler // handlerMap sync.Map
}

func NewHandlerMsg() *HandlerMgr {
	return &HandlerMgr{
		handlerMap: make(map[ProxyScope]map[HandlerKey]BaseHandler),
	}
}

// add handler to mgr
func (mgr *HandlerMgr) AddHandler(scope ProxyScope, key HandlerKey, base BaseHandler) {
	// check proto
	switch scope {
	case GlobalProxy, AppProxy:
	default:
		logger.Warningf("add handler proto not exist, proto: %v", scope)
		return
	}
	// add lock
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	// get base map
	baseMap := mgr.handlerMap[scope]
	// if not exist, create one
	if baseMap == nil {
		baseMap = make(map[HandlerKey]BaseHandler)
	}
	// check if handler already exist
	_, ok := baseMap[key]
	if ok {
		// if exist already, should ignore
		logger.Debugf("key has already in map, scope: %v, key: %v", scope, key)
		return
		//preBase.Close()
		//delete(baseMap, key)
	}
	// add handler
	baseMap[key] = base
	logger.Debugf("handler add to manager success, scope: %v, key: %v", scope, key)
}

// close and remove base handler
func (mgr *HandlerMgr) CloseBaseHandler(scope ProxyScope, key HandlerKey) {
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	baseMap, ok := mgr.handlerMap[scope]
	if !ok {
		return
	}
	base, ok := baseMap[key]
	if !ok {
		return
	}
	// close and delete
	base.Close()
	delete(baseMap, key)
}

// close handler according to proto
func (mgr *HandlerMgr) CloseProtoHandler(scope ProxyScope) {
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	baseMap, ok := mgr.handlerMap[scope]
	if !ok {
		return
	}
	// close handler
	for _, base := range baseMap {
		base.Close()
	}
	// delete proto handler
	delete(mgr.handlerMap, scope)
}

// close all handler
func (mgr *HandlerMgr) CloseAll() {
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	for proto, _ := range mgr.handlerMap {
		mgr.CloseProtoHandler(proto)
	}
}

// dial proxy
func DialProxy(proxy Config.Proxy) (net.Conn, error) {
	port := strconv.Itoa(proxy.Port)
	if port == "" {
		port = "80"
	}
	proxyAddr := proxy.Server + ":" + strconv.Itoa(proxy.Port)
	return net.DialTimeout("tcp", proxyAddr, 3*time.Second)
}

// conn communicate
func Communicate(local net.Conn, remote net.Conn) {
	// lConn to rConn
	go func() {
		logger.Debug("copy lConn -> rConn")
		_, err := io.Copy(local, remote)
		if err != nil {
			logger.Debugf("lConn to rConn closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
	// rConn to lConn
	go func() {
		logger.Debugf("copy rConn -> lConn")
		_, err := io.Copy(remote, local)
		if err != nil {
			logger.Debugf("rConn to lConn closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
}

func NewHandler(proto string, scope ProxyScope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) BaseHandler {
	// search proto
	switch proto {
	case "http":
		return NewHttpHandler(scope, key, proxy, lAddr, rAddr, lConn)
	case "sock4":
		return NewSock4Handler(scope, key, proxy, lAddr, rAddr, lConn)
	case "sock5tcp":
		return NewTcpSock5Handler(scope, key, proxy, lAddr, rAddr, lConn)
	case "sock5udp":
		return NewUdpSock5Handler(scope, key, proxy, lAddr, rAddr, lConn)
	default:
		logger.Warningf("unknown proto type: %v", proto)
	}
	return nil
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
