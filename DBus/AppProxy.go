package DBus

import (
	"fmt"

	com "github.com/DeepinProxy/Com"
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
		proxyPrv: initProxyPrv(define.App, define.AppPriority),
	}
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
	go func() {
		_ = mgr.addProxyApps(apps)
	}()
	return nil
}

func (mgr *AppProxy) addProxyApps(apps []string) error {
	// add app
	for _, app := range apps {
		// check if already exist
		if com.MegaExist(mgr.Proxies.ProxyProgram, app) {
			return nil
		}
		// controller
		err := mgr.controller.UpdateFromManager(app)
		if err != nil {
			return dbusutil.ToError(err)
		}
		mgr.Proxies.ProxyProgram = append(mgr.Proxies.ProxyProgram, app)
		// check if is in proxying
		if !mgr.Enabled {
			return nil
		}
		return nil
	}
	return nil
}

// delete proxy app
func (mgr *AppProxy) DelProxyApps(apps []string) *dbus.Error {
	go func() {
		_ = mgr.delProxyApps(apps)
	}()
	return nil
}

func (mgr *AppProxy) delProxyApps(apps []string) error {
	// add app
	for _, app := range apps {
		// check if already exist
		if !com.MegaExist(mgr.Proxies.ProxyProgram, app) {
			return nil
		}
		// controller
		err := mgr.controller.ReleaseToManager(app)
		if err != nil {
			return dbusutil.ToError(err)
		}
		ifc, _, err := com.MegaDel(mgr.Proxies.ProxyProgram, app)
		if err != nil {
			logger.Warningf("[%s] del proxy app %s failed, err: %v", mgr.scope, app, err)
			return err
		}
		temp, ok := ifc.([]string)
		if !ok {
			return nil
		}
		mgr.Proxies.ProxyProgram = temp
		return nil
	}
	return nil
}
