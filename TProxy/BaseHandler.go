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

// general method

type Communication struct {
}

func (c *Communication) Communicate(local net.Conn, remote net.Conn) {
	// local to remote
	go func() {
		logger.Debug("copy local -> remote")
		_, err := io.Copy(local, remote)
		if err != nil {
			logger.Debugf("local to remote closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
	// remote to local
	go func() {
		logger.Debugf("copy remote -> local")
		_, err := io.Copy(remote, local)
		if err != nil {
			logger.Debugf("remote to local closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
}

// handler private
type handlerPrv struct {
	local  net.Conn
	remote net.Conn
	key    HandlerKey
	proxy  Config.Proxy
	mgr    HandlerMgr
}

func newHandlerPrv(local net.Conn, remote net.Conn, key HandlerKey, proxy Config.Proxy) *handlerPrv {
	return &handlerPrv{
		local:  local,
		remote: remote,
		key:    key,
		proxy:  proxy,
	}
}

func (pr *handlerPrv) AddMgr() {

}

// handler module

type BaseHandler interface {
	Tunnel(rConn net.Conn, addr net.Addr) error
	Close()
}

type ProxyScope int

const (
	GlobalProxy ProxyScope = iota
	AppProxy
)

// handler key in case keep the same handler
type HandlerKey struct {
	SrcIP   string
	srcPort int
	DstIP   string
	DstPort int
}

// manager all handler
type HandlerMgr struct {
	handlerLock sync.RWMutex
	handlerMap  map[ProxyScope]map[HandlerKey]BaseHandler // handlerMap sync.Map
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
	preBase, ok := baseMap[key]
	if ok {
		preBase.Close()
		delete(baseMap, key)
	}
	baseMap[key] = base
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
	// local to remote
	go func() {
		logger.Debug("copy local -> remote")
		_, err := io.Copy(local, remote)
		if err != nil {
			logger.Debugf("local to remote closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
	// remote to local
	go func() {
		logger.Debugf("copy remote -> local")
		_, err := io.Copy(remote, local)
		if err != nil {
			logger.Debugf("remote to local closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
}

func NewHandler(local net.Conn, proxy Config.Proxy, proto string) BaseHandler {
	// search proto
	switch proto {
	case "http":
		return NewHttpHandler(local, proxy)
	case "sock4":
		return NewSock4Handler(local, proxy)
	case "sock5":
		return NewTcpSock5Handler(local, proxy)
	default:
		logger.Warningf("unknown proto type: %v", proto)
	}
	return nil
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
