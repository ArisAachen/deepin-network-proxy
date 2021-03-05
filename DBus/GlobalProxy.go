package DBus

import (
	"fmt"

	config "github.com/DeepinProxy/Config"
	define "github.com/DeepinProxy/Define"
	"github.com/godbus/dbus"
	"pkg.deepin.io/lib/dbusutil"
)

type GlobalProxy struct {
	proxyPrv

	// no proxy app
	IgnoreApp []string

	// methods
	methods *struct {
		ClearProxies func()
		SetProxies   func() `in:"proxies"`
		StartProxy   func() `in:"proto,name,udp" out:"err"`
		StopProxy    func()

		// diff method
		IgnoreProxyApps   func() `in:"app" out:"err"`
		UnIgnoreProxyApps func() `in:"app" out:"err"`
	}

	// signal
	signals *struct {
		Proxy struct {
			proxy config.Proxy
		}
	}
}

func NewGlobalProxy() *GlobalProxy {
	global := &GlobalProxy{
		proxyPrv: initProxyPrv(define.Global),
	}
	global.loadConfig()
	// _ = global.initCGroup()

	// _ = global.createTable()
	return global
}

func (mgr *GlobalProxy) export(service *dbusutil.Service) error {
	if service == nil {
		logger.Warningf("[%s] export service is nil", mgr.scope)
		return fmt.Errorf("[%s] export service is nil", mgr.scope)
	}
	err := service.Export(mgr.getDBusPath(), mgr)
	if err != nil {
		logger.Warningf("[%s] export service failed, err: %v", mgr.scope, err)
		return err
	}
	return nil
}

// add proxy app
func (mgr *GlobalProxy) IgnoreProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.addCGroupExes(apps)
	return nil
}

// delete proxy app
func (mgr *GlobalProxy) UnIgnoreProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.delCGroupExes(apps)
	return nil
}

