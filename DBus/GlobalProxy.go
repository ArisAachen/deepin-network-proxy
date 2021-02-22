package DBus

import (
	"fmt"
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
		IgnoreProxy   func() `in:"app" out:"err"`
		UnIgnoreProxy func() `in:"app" out:"err"`
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
	return global
}

func (mgr *GlobalProxy) export(service *dbusutil.Service) error {
	if service == nil {
		logger.Warningf("[%s] export service is nil", mgr.getScope())
		return fmt.Errorf("[%s] export service is nil", mgr.getScope())
	}
	err := service.Export(mgr.getDBusPath(), mgr)
	if err != nil {
		logger.Warningf("[%s] export service failed, err: %v", mgr.getScope(), err)
		return err
	}
	return nil
}

// rewrite get scope
func (mgr *GlobalProxy) getScope() tProxy.ProxyScope {
	return tProxy.GlobalProxy
}

// rewrite export DBus path
func (mgr *GlobalProxy) getDBusPath() dbus.ObjectPath {
	path := BusPath + "/" + tProxy.GlobalProxy.String()
	return dbus.ObjectPath(path)
}

// add proxy app
func (mgr *GlobalProxy) IgnoreProxy(app []string) *dbus.Error {
	return nil
}

// delete proxy app
func (mgr *GlobalProxy) UnIgnoreProxy(app []string) *dbus.Error {
	return nil
}
