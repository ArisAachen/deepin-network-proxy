package DBus

import (
	"sync"

	cgroup "github.com/DeepinProxy/CGroups"
	config "github.com/DeepinProxy/Config"
	define "github.com/DeepinProxy/Define"
	newCGroups "github.com/DeepinProxy/NewCGroups"
	newIptables "github.com/DeepinProxy/NewIptables"
	tProxy "github.com/DeepinProxy/TProxy"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

// use to init proxy onceCfg
var allProxyCfg *config.ProxyConfig
var onceCfg sync.Once

const (
	BusServiceName = "com.deepin.session.proxy"
	BusPath        = "/com/deepin/session/proxy"
	BusInterface   = BusServiceName
)

// must ignore proxy proc
var mainProxy []string = []string{
	"DeepinProxy",
	"Qv2ray",
}

type proxyPrv struct {
	scope    define.Scope
	priority define.Priority

	// proxy message
	Proxies config.ScopeProxies
	Proxy   config.Proxy // current proxy

	// if proxy opened
	Enabled bool

	// handler manager
	manager *Manager

	// proxyMember to organize cgroup v2
	cgroupMember *cgroup.CGroupMember

	// cgroup controller
	controller *newCGroups.Controller

	// iptables chain rule slice[2]
	chains [2]*newIptables.Chain

	// handler manager
	handlerMgr *tProxy.HandlerMgr

	// stop chan
	stop chan bool
}

// init proxy private
func initProxyPrv(scope define.Scope, priority define.Priority) proxyPrv {
	return proxyPrv{
		scope:      scope,
		priority:   priority,
		handlerMgr: tProxy.NewHandlerMsg(scope),
		Proxies: config.ScopeProxies{
			Proxies:      make(map[string][]config.Proxy),
			ProxyProgram: make([]string, 10),
			WhiteList:    make([]string, 10),
		},
	}
}

// load config from user home dir
func (mgr *proxyPrv) loadConfig() {
	// load proxy from manager
	mgr.Proxies, _ = mgr.manager.config.GetScopeProxies(mgr.scope)
	logger.Debugf("[%s] load config success, config: %v", mgr.scope, mgr.Proxies)
}

func (mgr *proxyPrv) saveManager(manager *Manager) {
	mgr.manager = manager
}

// write config
func (mgr *proxyPrv) writeConfig() error {
	// set and write config
	mgr.manager.config.SetScopeProxies(mgr.scope, mgr.Proxies)
	err := mgr.manager.WriteConfig()
	if err != nil {
		logger.Warning("[%s] write config failed, err:%v", mgr.scope, err)
		return err
	}
	return nil
}
