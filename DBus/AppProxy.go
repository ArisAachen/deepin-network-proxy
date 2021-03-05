package DBus

import (
	"fmt"

	config "github.com/DeepinProxy/Config"
	define "github.com/DeepinProxy/Define"
	"github.com/godbus/dbus"
	"pkg.deepin.io/lib/dbusutil"
)

type AppProxy struct {
	proxyPrv

	// methods
	methods *struct {
		ClearProxies func()
		SetProxies   func() `in:"proxies"`
		StartProxy   func() `in:"proto,name,udp" out:"err"`
		StopProxy    func()

		// diff method
		AddProxyApps func() `in:"app" out:"err"`
		DelProxyApps func() `in:"app" out:"err"`
	}

	// signal
	signals *struct {
		Proxy struct {
			proxy config.Proxy
		}
	}
}

// create app proxy
func NewAppProxy() *AppProxy {
	appModule := &AppProxy{
		proxyPrv: initProxyPrv(define.App),
	}
	// load config
	appModule.loadConfig()


	apps := appModule.Proxies.ProxyProgram
	appModule.addCGroupExes(apps)

	// init iptables
	appModule.createTable()

	return appModule
}

func (mgr *AppProxy) export(service *dbusutil.Service) error {
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
func (mgr *AppProxy) AddProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.addCGroupExes(apps)
	return nil
}

// delete proxy app
func (mgr *AppProxy) DelProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.delCGroupExes(apps)
	return nil
}
