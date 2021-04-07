package TProxy

import (
	"fmt"
	"net"
	"sync"

	define "github.com/DeepinProxy/Define"
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

	// write and read
	WriteRemote([]byte) error
	WriteLocal([]byte) error
	ReadRemote([]byte) error
	ReadLocal([]byte) error
	Communicate()
}

// proto
type ProtoTyp string

const (
	NoneProto ProtoTyp = "no-proto"
	HTTP      ProtoTyp = "http"
	SOCK4     ProtoTyp = "sock4"
	SOCK5TCP  ProtoTyp = "sock5-tcp"
	SOCK5UDP  ProtoTyp = "sock5-udp"
)

func BuildProto(proto string) (ProtoTyp, error) {
	switch proto {
	case "no-proxy":
		return NoneProto, nil
	case "http":
		return HTTP, nil
	case "sock4":
		return SOCK4, nil
	case "sock5-tcp":
		return SOCK5TCP, nil
	case "sock5-udp":
		return SOCK5UDP, nil
	default:
		return NoneProto, fmt.Errorf("scope is invalid, scope: %v", proto)
	}
}

func (Typ ProtoTyp) String() string {
	switch Typ {
	case NoneProto:
		return "no-proxy"
	case HTTP:
		return "http"
	case SOCK4:
		return "sock4"
	case SOCK5TCP:
		return "sock5-tcp"
	case SOCK5UDP:
		return "sock5-udp"
	default:
		return "unknown-proto"
	}
}

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
	handlerLock sync.Mutex
	handlerMap  map[ProtoTyp]map[HandlerKey]BaseHandler // handlerMap sync.Map map[http udp]map[HandlerKey]BaseHandler

	// scope [global,app]
	scope define.Scope
	// chan to stop accept
	stop chan bool
}

func NewHandlerMgr(scope define.Scope) *HandlerMgr {
	return &HandlerMgr{
		scope:      scope,
		handlerMap: make(map[ProtoTyp]map[HandlerKey]BaseHandler),
		stop:       make(chan bool),
	}
}

// add handler to mgr
func (mgr *HandlerMgr) AddHandler(typ ProtoTyp, key HandlerKey, base BaseHandler) {
	// add lock
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	// check if handler already exist
	baseMap, ok := mgr.handlerMap[typ]
	if !ok {
		baseMap = make(map[HandlerKey]BaseHandler)
		mgr.handlerMap[typ] = baseMap
	}
	_, ok = baseMap[key]
	if ok {
		// if exist already, should ignore
		logger.Debugf("[%s] key has already in map, type: %v, key: %v", mgr.scope, typ, key)
		return
	}
	// add handler
	baseMap[key] = base
	logger.Debugf("[%s] handler add to manager success, type: %v, key: %v", mgr.scope, typ, key)
}

// close and remove base handler
func (mgr *HandlerMgr) CloseBaseHandler(typ ProtoTyp, key HandlerKey) {
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	baseMap, ok := mgr.handlerMap[typ]
	if !ok {
		logger.Debugf("[%s] delete base map dont exist in map", mgr.scope)
		return
	}
	base, ok := baseMap[key]
	if !ok {
		logger.Debugf("[%s] delete key dont exist in base map, key: %v", mgr.scope, key)
		return
	}
	// close and delete
	base.Close()
	delete(baseMap, key)
	logger.Debugf("[%s] delete key successfully, key: %v", mgr.scope, key)
}

// close handler according to proto
func (mgr *HandlerMgr) CloseTypHandler(typ ProtoTyp) {
	mgr.handlerLock.Lock()
	defer mgr.handlerLock.Unlock()
	baseMap, ok := mgr.handlerMap[typ]
	if !ok {
		return
	}
	// close handler
	for _, base := range baseMap {
		base.Close()
	}
	// delete proto handler
	delete(mgr.handlerMap, typ)
}

// close all handler
func (mgr *HandlerMgr) CloseAll() {
	for proto, _ := range mgr.handlerMap {
		mgr.CloseTypHandler(proto)
	}
}

func NewHandler(proto ProtoTyp, scope define.Scope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) BaseHandler {
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
	logger.SetLogLevel(log.LevelInfo)
}
