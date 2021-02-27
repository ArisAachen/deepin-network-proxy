package DBus

import (
	"fmt"

	cgroup "github.com/DeepinProxy/CGroups"
	config "github.com/DeepinProxy/Config"
	tProxy "github.com/DeepinProxy/TProxy"
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
		proxyPrv: initProxyPrv(tProxy.GlobalProxy),
	}
	global.loadConfig()
	_ = global.initCGroup()
	return global
}

func (mgr *GlobalProxy) export(service *dbusutil.Service) error {
	if service == nil {
		logger.Warningf("[%s] export service is nil", mgr.scope.String())
		return fmt.Errorf("[%s] export service is nil", mgr.scope.String())
	}
	err := service.Export(mgr.getDBusPath(), mgr)
	if err != nil {
		logger.Warningf("[%s] export service failed, err: %v", mgr.scope.String(), err)
		return err
	}
	return nil
}

func (mgr *GlobalProxy) initCGroup() error {
	// will not error in any case
	err := mgr.proxyPrv.initCGroup()
	if err != nil {
		return err
	}
	// make dir
	member, err := allCGroups.CreateCGroup(cgroup.GlobalProxyLevel, mgr.scope.String())
	if err != nil {
		return err
	}
	mgr.cgroupMember = member
	logger.Debugf("[%s] create cgroup success", mgr.scope.String())
	return nil
}

// rewrite get scope
func (mgr *GlobalProxy) getScope() tProxy.ProxyScope {
	return tProxy.GlobalProxy
}

func (mgr *GlobalProxy) getCGroupLevel() int {
	return cgroup.GlobalProxyLevel
}

// rewrite export DBus path
func (mgr *GlobalProxy) getDBusPath() dbus.ObjectPath {
	path := BusPath + "/" + tProxy.GlobalProxy.String()
	return dbus.ObjectPath(path)
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
