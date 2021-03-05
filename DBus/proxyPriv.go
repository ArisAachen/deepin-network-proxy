package DBus

import (
	"os"
	"sync"

	cgroup "github.com/DeepinProxy/CGroups"
	com "github.com/DeepinProxy/Com"
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
func initProxyPrv(scope define.Scope) proxyPrv {
	return proxyPrv{
		scope:      scope,
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

// write config
func (mgr *proxyPrv) writeConfig() error {
	// get config path
	path, err := com.GetUserConfigDir()
	if err != nil {
		logger.Warningf("[%s] get user home dir failed, user:%v, err: %v", mgr.scope, os.Geteuid(), err)
		return err
	}
	// check if all proxy is legal
	if allProxyCfg == nil {
		return err
	}
	// set and write config
	allProxyCfg.SetScopeProxies(mgr.scope, mgr.Proxies)
	err = allProxyCfg.WritePxyCfg(path)
	if err != nil {
		logger.Warningf("[%s] write config file failed, err: %v", mgr.scope, err)
		return err
	}
	return nil
}

// add cgroup proc
func (mgr *proxyPrv) addCGroupExes(exes []string) {
	mgr.cgroupMember.AddTgtExes(exes)
	//allCGroups.AddCGroupProcs(elem, procs)
}

// add cgroup proc
func (mgr *proxyPrv) delCGroupExes(exes []string) {
	mgr.cgroupMember.DelTgtExes(exes, true)
	//allCGroups.DelCGroupProcs(elem, procs)
}
