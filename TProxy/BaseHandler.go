package TProxy

import (
	"net"
	"sync"

	"github.com/DeepinProxy/Config"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

// handler module

type BaseHandler interface {
	// connection
	Tunnel() error

	// close
	Close()  // direct close handler
	Remove() // remove self from map
	AddMgr(mgr *HandlerMgr)

	// write adn read
	WriteRemote([]byte) error
	WriteLocal([]byte) error
	ReadRemote([]byte) error
	ReadLocal([]byte) error
	Communicate()
}

type ProxyScope int

const (
	NoneProxy ProxyScope = iota
	GlobalProxy
	AppProxy
)

func (scope ProxyScope) String() string {
	switch scope {
	case NoneProxy:
		return "no-proxy"
	case GlobalProxy:
		return "global-proxy"
	case AppProxy:
		return "app-proxy"
	default:
		return "scope-type error"
	}
}

type ProxyTyp string

const (
	NoneTyp  ProxyTyp = "no-proxy"
	HTTP     ProxyTyp = "http"
	SOCK4    ProxyTyp = "sock4"
	SOCK5TCP ProxyTyp = "sock5-tcp"
	SOCK5UDP ProxyTyp = "sock5-udp"
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
func (mgr *HandlerMgr) AddHandler(typ ProxyTyp, scope ProxyScope, key HandlerKey, base BaseHandler) {
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
		mgr.handlerMap[scope] = baseMap
	}
	// check if handler already exist
	_, ok := baseMap[key]
	if ok {
		// if exist already, should ignore
		logger.Debugf("[%s] key has already in map, type: %v, key: %v", scope.String(), typ, key)
		return
		//preBase.Close()
		//delete(baseMap, key)
	}
	// add handler
	baseMap[key] = base
	logger.Debugf("[%s] handler add to manager success, type: %v, key: %v", scope.String(), typ, key)
}

// close and remove base handler
func (mgr *HandlerMgr) CloseBaseHandler(scope ProxyScope, key HandlerKey) {
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	baseMap, ok := mgr.handlerMap[scope]
	if !ok {
		logger.Debugf("[%s] delete base map dont exist in map", scope.String())
		return
	}
	base, ok := baseMap[key]
	if !ok {
		logger.Debugf("[%s] delete key dont exist in base map, key: %v", scope.String(), key)
		return
	}
	// close and delete
	base.Close()
	delete(baseMap, key)
	logger.Debugf("[%s] delete key successfully, key: %v", scope.String(), key)
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

func NewHandler(proto ProxyTyp, scope ProxyScope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) BaseHandler {
	// search proto
	switch proto {
	case HTTP:
		return NewHttpHandler(scope, key, proxy, lAddr, rAddr, lConn)
	case SOCK4:
		return NewSock4Handler(scope, key, proxy, lAddr, rAddr, lConn)
	case SOCK5TCP:
		return NewTcpSock5Handler(scope, key, proxy, lAddr, rAddr, lConn)
	case SOCK5UDP:
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
