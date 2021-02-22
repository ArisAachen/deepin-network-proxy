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
		AddProxy func() `in:"app" out:"err"`
		DelProxy func() `in:"app" out:"err"`
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
	return app
}

func (mgr *AppProxy) export(service *dbusutil.Service) error {
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
func (mgr *AppProxy) getScope() tProxy.ProxyScope {
	return tProxy.AppProxy
}

// rewrite export DBus path
func (mgr *AppProxy) getDBusPath() dbus.ObjectPath {
	path := BusPath + "/" + tProxy.AppProxy.String()
	return dbus.ObjectPath(path)
}

// add proxy app
func (mgr *AppProxy) AddProxy(app []string) *dbus.Error {
	return nil
}

// delete proxy app
func (mgr *AppProxy) DelProxy(app []string) *dbus.Error {
	return nil
}
