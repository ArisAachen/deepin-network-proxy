package DBus

import (
	"fmt"
	com "github.com/DeepinProxy/Com"

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
		proxyPrv: initProxyPrv(define.Global, define.GlobalPriority),
	}
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
	go func() {
		_ = mgr.ignoreProxyApps(apps)
	}()
	return nil
}

func (mgr *GlobalProxy) ignoreProxyApps(apps []string) error {
	// add app
	for _, app := range apps {
		// check if already exist
		if com.MegaExist(mgr.Proxies.NoProxyProgram, app) {
			return nil
		}
		mgr.Proxies.NoProxyProgram = append(mgr.Proxies.NoProxyProgram, app)
		_ = mgr.writeConfig()
		// check if is in proxying
		if !mgr.Enabled {
			return nil
		}
		// controller
		err := mgr.controller.UpdateFromManager(app)
		if err != nil {
			return dbusutil.ToError(err)
		}
		return nil
	}
	return nil
}

// delete proxy app
func (mgr *GlobalProxy) UnIgnoreProxyApps(apps []string) *dbus.Error {
	go func() {
		_ = mgr.unIgnoreProxyApps(apps)
	}()
	return nil
}

func (mgr *GlobalProxy) unIgnoreProxyApps(apps []string) error {
	// add app
	for _, app := range apps {
		// check if already exist
		if !com.MegaExist(mgr.Proxies.NoProxyProgram, app) {
			return nil
		}
		ifc, _, err := com.MegaDel(mgr.Proxies.NoProxyProgram, app)
		if err != nil {
			logger.Warningf("[%s] del proxy app %s failed, err: %v", mgr.scope, app, err)
			return err
		}
		temp, ok := ifc.([]string)
		if !ok {
			return nil
		}
		mgr.Proxies.NoProxyProgram = temp
		_ = mgr.writeConfig()
		if !mgr.Enabled {
			return nil
		}
		// controller
		err = mgr.controller.ReleaseToManager(app)
		if err != nil {
			return dbusutil.ToError(err)
		}
		return nil
	}
	return nil
}
