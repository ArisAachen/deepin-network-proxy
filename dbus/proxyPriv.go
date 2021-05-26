package DBus

import (
	com "github.com/ArisAachen/deepin-network-proxy/com"
	config "github.com/ArisAachen/deepin-network-proxy/config"
	define "github.com/ArisAachen/deepin-network-proxy/define"
	"github.com/ArisAachen/deepin-network-proxy/ip_route"
	newCGroups "github.com/ArisAachen/deepin-network-proxy/new_cgroups"
	newIptables "github.com/ArisAachen/deepin-network-proxy/new_iptables"
	tProxy "github.com/ArisAachen/deepin-network-proxy/tproxy"
	"net"
	"path/filepath"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

const (
	BusServiceName = "com.deepin.system.proxy"
	BusPath        = "/com/deepin/system/proxy"
	BusInterface   = BusServiceName
)

// must ignore proxy proc
var mainProxy = []string{
	"/usr/lib/deepin-daemon/dde-proxy",
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

	// listener
	tcpHandler net.Listener
	udpHandler net.PacketConn

	// cgroup controller
	controller *newCGroups.Controller

	// iptables chain rule slice[2]
	chains [2]*newIptables.Chain

	// route rule
	ipRule *IpRoute.Rule

	// handler manager
	handlerMgr *tProxy.HandlerMgr

	// stop chan
	stop bool
}

// init proxy private
func initProxyPrv(scope define.Scope, priority define.Priority) proxyPrv {
	prv := proxyPrv{
		scope:      scope,
		priority:   priority,
		handlerMgr: tProxy.NewHandlerMgr(scope),
		stop:       true,
		Proxies: config.ScopeProxies{
			Proxies:      make(map[string][]config.Proxy),
			ProxyProgram: []string{},
			WhiteList:    []string{},
		},
	}
	return prv
}

// proxy prepare
func (mgr *proxyPrv) startRedirect() error {
	// make sure manager start init
	mgr.manager.Start()

	// clean old redirect
	_ = mgr.firstClean()

	// create cgroups
	err := mgr.createCGroupController()
	if err != nil {
		logger.Warning("[%s] create cgroup failed, err: %v", err)
		return err
	}

	// create iptables
	err = mgr.createTable()
	if err != nil {
		logger.Warning("[%s] create iptables failed, err: %v", err)
		return err
	}
	err = mgr.appendRule()
	if err != nil {
		logger.Warning("[%s] append iptables failed, err: %v", mgr.scope, err)
		return err
	}

	err = mgr.createIpRule()
	if err != nil {
		logger.Warning("[%s] create ip rule failed, err: %v", err)
		return err
	}
	logger.Debugf("[%s] start tproxy iptables cgroups ipRule success", mgr.scope)

	//// first adjust cgroups
	//err = mgr.firstAdjustCGroups()
	//if err != nil {
	//	logger.Warningf("[%s] first adjust controller failed, err: %v", mgr.scope, err)
	//	return err
	//}
	//logger.Debugf("[%s] first adjust controller success", mgr.scope)
	return nil
}

//
func (mgr *proxyPrv) stopRedirect() error {
	// release iptables rules
	err := mgr.releaseRule()
	if err != nil {
		logger.Warningf("[%s] release iptables failed, err: %v", mgr.scope, err)
		return err
	}

	// release cgroups
	err = mgr.releaseController()
	if err != nil {
		logger.Warningf("[%s] release controller failed, err: %v", mgr.scope, err)
		return err
	}

	err = mgr.releaseIpRule()
	if err != nil {
		logger.Warningf("[%s] release ipRule failed, err: %v", mgr.scope, err)
	}

	// try to release manager
	err = mgr.manager.release()
	if err != nil {
		logger.Warningf("[%s] release manager failed, err: %v", mgr.scope, err)
		return err
	}

	logger.Debugf("[%s] stop tproxy iptables cgroups ipRule success", mgr.scope)
	return nil
}

// load config
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

// first clean
func (mgr *proxyPrv) firstClean() error {
	// get config path
	path, err := com.GetConfigDir()
	if err != nil {
		logger.Warningf("[%s] run first clean failed, config err: %v", mgr.scope, err)
		return err
	}
	// get script file path
	path = filepath.Join(path, define.ScriptName)
	// run script
	buf, err := com.RunScript(path, []string{"clear_" + mgr.scope.String()})
	if err != nil {
		logger.Debugf("[%s] run first clean script failed, out: %s, err: %v", mgr.scope, string(buf), err)
		return err
	}
	logger.Debugf("[%s] run first clean script success", mgr.scope)
	return nil
}
