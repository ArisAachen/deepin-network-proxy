package DBus

import (
	"fmt"
	com "github.com/ArisAachen/deepin-network-proxy/com"
	"os"

	config "github.com/ArisAachen/deepin-network-proxy/config"
	define "github.com/ArisAachen/deepin-network-proxy/define"
	"github.com/godbus/dbus"
	"pkg.deepin.io/lib/dbusutil"
)

type GlobalProxy struct {
	proxyPrv

	// no proxy app
	IgnoreApp []string

	// methods
	methods *struct {
		ClearProxy func()
		SetProxies func() `in:"proxies" out:"err"`
		StartProxy func() `in:"proto,name,udp" out:"err"`
		StopProxy  func()
		GetProxy   func() `out:"proxy"`
		AddProxy   func() `in:"proto,name,proxy"`
		GetCGroups func() `out:"cgroups"`

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
	// get all procs message
	procsMap, err := mgr.manager.GetAllProcs()
	if err != nil {
		return err
	}

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
			err := mgr.controller.UpdateFromManager(app)
			if err != nil {
				logger.Warningf("[%s] add proc %s from %s at add proxy apps failed, err: %v", mgr.scope, app, controller.Name, err)
			} else {
				logger.Debugf("[%s] add proc %s from %s at add proxy apps success", mgr.scope, app, controller.Name)
			}
			mgr.controller.AddCtlAppPath(app)
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

// cgroups
func (mgr *GlobalProxy) GetCGroups() (string, *dbus.Error) {
	path := "/sys/fs/cgroup/unified/Global.slice/cgroups.procs"
	_, err := os.Stat(path)
	if err != nil {
		logger.Warningf("app cgroups not exist, err: %v", err)
		return "", dbusutil.ToError(err)
	}
	return path, nil
}