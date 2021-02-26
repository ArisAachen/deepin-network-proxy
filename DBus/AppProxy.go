package DBus

import (
	"fmt"

	config "github.com/DeepinProxy/Config"
	tProxy "github.com/DeepinProxy/TProxy"
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
	app := &AppProxy{
		proxyPrv: initProxyPrv(tProxy.AppProxy),
	}
	app.loadConfig()
	_ = app.initCGroup()
	return app
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

// rewrite get scope
func (mgr *AppProxy) getScope() tProxy.ProxyScope {
	return tProxy.AppProxy
}

// rewrite export DBus path
func (mgr *AppProxy) getDBusPath() dbus.ObjectPath {
	path := BusPath + "/" + tProxy.AppProxy.String()
	return dbus.ObjectPath(path)
}

// init cgroup
func (mgr *AppProxy) initCGroup() error {
	// will not error in any case
	err := mgr.proxyPrv.initCGroup()
	if err != nil {
		return err
	}
	// make dir
	err = allCGroups.CreateCGroup(2, mgr.scope.String())
	if err != nil {
		return err
	}
	logger.Debugf("[%s] create cgroup success", mgr.scope)
	return nil
}

// add proxy app
func (mgr *AppProxy) AddProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.addCGroupProcs(mgr.scope.String(), apps)
	return nil
}

// delete proxy app
func (mgr *AppProxy) DelProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.delCGroupProcs(mgr.scope.String(), apps)
	return nil
}
