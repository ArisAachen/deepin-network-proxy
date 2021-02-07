package main

import (
	"errors"
	"os"
	"sync"

	com "github.com/DeepinProxy/Com"
	config "github.com/DeepinProxy/Config"
	tProxy "github.com/DeepinProxy/TProxy"
)

// interface and path name
const (
	DBusServiceName = "com.deepin.system.proxy"
	DBusPath        = "/com/deepin/system/proxy"
	DBusInterface   = DBusServiceName
)

type ProxyManager struct {
	// proxy message
	Proxies config.ProxyConfig
	Proxy   config.Proxy // current proxy

	// if proxy opened
	Enabled bool

	// signal
	signals *struct {
		Enabled bool
		Proxy   config.Proxy
	}

	// handler manager
	globalSignal sync.Cond
	appSignal    sync.Cond
}

// create proxy manager
func NewProxyManager() *ProxyManager {
	// load config
	manager := new(ProxyManager)
	manager.loadConfig()
	return manager
}

// load config from user home dir
func (mgr *ProxyManager) loadConfig() {
	// get effective user config dir
	path, err := com.GetUserConfigDir()
	if err != nil {
		logger.Warningf("failed to get user home dir, user:%v, err: %v", os.Geteuid(), err)
		return
	}
	err = mgr.Proxies.LoadPxyCfg(path)
	if err != nil {
		logger.Warningf("load config failed, path: %s, err: %v", path, err)
		return
	}
}

// write config
func (mgr *ProxyManager) writeConfig() {
	path, err := com.GetUserConfigDir()
	if err != nil {
		logger.Warningf("failed to get user home dir, user:%v, err: %v", os.Geteuid(), err)
		return
	}
	err = mgr.Proxies.WritePxyCfg(path)
	if err != nil {
		logger.Warningf("load config failed, path: %s, err: %v", path, err)
		return
	}
}

// implement interface
func (mgr *ProxyManager) GetInterfaceName() string {
	return DBusInterface
}

// clear all proxies
func (mgr *ProxyManager) ClearProxies() {
	mgr.Proxies = config.ProxyConfig{}
	mgr.writeConfig()
}

// reset all proxy
func (mgr *ProxyManager) SetProxies(proxies config.ProxyConfig) {
	// if proxies include globalSignal proxy, stop and reset globalSignal proxy
	if global, ok := proxies.AllProxies[tProxy.GlobalProxy.String()]; ok {
		logger.Debugf("change global proxy config, proxies: %v", proxies)
		mgr.StopProxy(tProxy.GlobalProxy.String())
		mgr.Proxies.AllProxies[tProxy.GlobalProxy.String()] = global
	}
	// if proxies include appSignal proxy, stop and reset appSignal proxy
	if app, ok := proxies.AllProxies[tProxy.AppProxy.String()]; ok {
		logger.Debugf("change appSignal proxy config, proxies: %v", proxies)
		mgr.StopProxy(tProxy.AppProxy.String())
		mgr.Proxies.AllProxies[tProxy.AppProxy.String()] = app
	}
	// write config
	mgr.writeConfig()
}

// start proxy   [globalSignal,appSignal] -> [http,sock4,sock5] -> [http_one] -> false
func (mgr *ProxyManager) StartProxy(scope string, lsp string, proto string, name string, udp bool) error {
	proxy, err := mgr.Proxies.GetProxy(scope, proto, name)
	if err != nil {
		logger.Warningf("search proxy failed, err: %v", err)
		return err
	}

	// start proxy
	return mgr.startProxy(scope, lsp, proto, proxy, udp)
}

func (mgr *ProxyManager) startProxy(scope string, lsp string, proto string, proxy config.Proxy, udp bool) error {
	// make scope
	tScope, err := tProxy.BuildScope(scope)
	if err != nil {
		return err
	}

	// broadcast stop signal
	var broadcaster *sync.Cond
	// check proxy type
	switch tScope {
	case tProxy.GlobalProxy:
		broadcaster = &mgr.globalSignal
	case tProxy.AppProxy:
		broadcaster = &mgr.appSignal
	default:
		return errors.New("scope dont exist")
	}

	var tProto tProxy.ProtoTyp
	if proto == "sock5" {
		// never err
		tProto = tProxy.SOCK5TCP
	} else {
		tProto, err = tProxy.BuildProto(proto)
		if err != nil {
			return err
		}
	}
	// create new tcp proxy
	go NewTcpProxy(tScope, lsp, tProto, proxy, broadcaster)
	// check if need create udp proxy
	if proto == "sock5" && udp {
		go NewUdpProxy(tScope, lsp, proxy, broadcaster)
	}

	return nil
}

// stop proxy according to proxy type
func (mgr *ProxyManager) StopProxy(scope string) {
	switch scope {
	case tProxy.GlobalProxy.String():
		// stop global proxy
		logger.Debug("stop global proxy")
		// broadcast signal to terminal global proxy
		mgr.globalSignal.Broadcast()
	case tProxy.AppProxy.String():
		// stop app proxy
		logger.Debug("stop appSignal proxy")
		// broadcast signal to terminal app proxy
		mgr.appSignal.Broadcast()
	default:
		logger.Warningf("stop proxy error, proxy type not exist, type: %v", scope)
	}
}
