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
		SetProxies   func() `in:"proxies" out:"err"`
		StartProxy   func() `in:"proto,name,udp" out:"err"`
		StopProxy    func()
		GetProxy     func() `out:"proxy"`
		AddProxy     func() `in:"proto,name,proxy"`

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
	// get all procs message
	procsMap, err := mgr.manager.GetAllProcs()
	if err != nil {
		return err
	}

	// add app
	for _, app := range apps {
		// check if already exist
		if com.MegaExist(mgr.Proxies.ProxyProgram, app) {
			return nil
		}
		mgr.Proxies.ProxyProgram = append(mgr.Proxies.ProxyProgram, app)
		// check if is in proxying
		if !mgr.Enabled {
			return nil
		}
		_ = mgr.writeConfig()
		// controller

		// get origin controller
		controller := mgr.manager.controllerMgr.GetControllerByCtlPath(app)
		if controller == nil {
			// add path
			mgr.controller.AddCtlAppPath(app)
			// get proc message
			procSl, ok := procsMap[app]
			if !ok {
				continue
			}
			// if not empty, move in
			err := mgr.controller.MoveIn(app, procSl)
			if err != nil {
				logger.Warningf("[%s] add procs %s at add proxy apps failed, err: %v", mgr.scope, app, err)
				continue
			}
			logger.Debugf("[%s] add procs %s at add proxy apps success", mgr.scope, app)
		} else {
			err = mgr.controller.UpdateFromManager(app)
			if err != nil {
				logger.Warningf("[%s] add proc %s from %s at add proxy apps failed, err: %v", mgr.scope, app, controller.Name, err)
			} else {
				logger.Debugf("[%s] add proc %s from %s at add proxy apps success", mgr.scope, app, controller.Name)
			}
			mgr.controller.AddCtlAppPath(app)
		}

		//err := mgr.controller.UpdateFromManager(app)
		//if err != nil {
		//	return dbusutil.ToError(err)
		//}
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
		// mega del
		ifc, _, err := com.MegaDel(mgr.Proxies.ProxyProgram, app)
		if err != nil {
			logger.Warningf("[%s] del proxy app %s failed, err: %v", mgr.scope, app, err)
			return err
		}
		temp, ok := ifc.([]string)
		if !ok && ifc != nil {
			return nil
		}
		mgr.Proxies.ProxyProgram = temp
		_ = mgr.writeConfig()
		// controller
		err = mgr.controller.ReleaseToManager(app)
		if err != nil {
			return dbusutil.ToError(err)
		}
		return nil
	}
	return nil
}
