package DBus

import (
	"os"
	"pkg.deepin.io/lib/dbusutil"
	"sync"

	com "github.com/DeepinProxy/Com"
	config "github.com/DeepinProxy/Config"
	"github.com/godbus/dbus"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

// use to init proxy once
var allProxy *config.ProxyConfig
var once sync.Once

type baseProxy struct {
	// proxy message
	Proxies config.ScopeProxies
	Proxy   config.Proxy // current proxy

	// if proxy opened
	Enabled bool

	// methods
	methods *struct {
		ClearProxies func()
		SetProxies   func() `in:"proxies"`
		StartProxy   func() `in:"proto,name,udp" out:"err"`
		StopProxy    func() `in:"scope"`
	}

	// signal
	signals *struct {
		Proxy struct {
			proxy config.Proxy
		}
	}
}

// load config from user home dir
func (mgr *baseProxy) loadConfig() {
	// get effective user config dir
	path, err := com.GetUserConfigDir()
	if err != nil {
		logger.Warningf("failed to get user home dir, user:%v, err: %v", os.Geteuid(), err)
		return
	}

	// init proxy once
	once.Do(func() {
		// load proxy
		allProxy = config.NewProxyCfg()
		err = allProxy.LoadPxyCfg(path)
		if err != nil {
			logger.Warningf("load config failed, path: %s, err: %v", path, err)
			return
		}
	})
	// get proxies from config
	mgr.Proxies, err = allProxy.GetScopeProxies(mgr.getScope())
	if err != nil {
		logger.Warningf("[%s] get proxies from global proxies failed, err: %v", mgr.getScope(), err)
		return
	}
}

// write config
func (mgr *baseProxy) writeConfig() {
	// get config path
	path, err := com.GetUserConfigDir()
	if err != nil {
		logger.Warningf("[%s] get user home dir failed, user:%v, err: %v", mgr.getScope(), os.Geteuid(), err)
		return
	}
	// check if all proxy is legal
	if allProxy == nil {
		return
	}
	// set and write config
	allProxy.SetScopeProxies(mgr.getScope(), mgr.Proxies)
	err = allProxy.WritePxyCfg(path)
	if err != nil {
		logger.Warningf("[%s] write config file failed, err: %v", mgr.getScope(), err)
		return
	}
}

// global and app rewrite this method to diff
func (mgr *baseProxy) getScope() string {
	return "base"
}

func (mgr *baseProxy) StartProxy(proto string, name string, udp bool) *dbus.Error {
	// get proxies
	proxy, err := mgr.Proxies.GetProxy(proto, name)
	if err != nil {
		logger.Warningf("[%s] get proxy failed, err: %v", mgr.getScope(), err)
		return dbusutil.ToError(err)
	}

	return nil
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
