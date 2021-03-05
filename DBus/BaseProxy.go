package DBus

import (
	config "github.com/DeepinProxy/Config"
	define "github.com/DeepinProxy/Define"
	"github.com/godbus/dbus"
	"pkg.deepin.io/lib/dbusutil"
	"pkg.deepin.io/lib/log"
)

// https://www.kernel.org/doc/Documentation/networking/tproxy.txt

type BaseProxy interface {
	// DBus method
	StartProxy(proto string, name string, udp bool) *dbus.Error
	StopProxy()
	SetProxies(proxies config.ScopeProxies) *dbus.Error
	ClearProxies() *dbus.Error

	// getScope() tProxy.ProxyScope
	getDBusPath() dbus.ObjectPath
	getScope() define.Scope

	// get cgroup v2 level
	getCGroupLevel() int

	// cgroup v2
	initCGroup() error
	addCGroupExes(procs []string)
	delCGroupExes(procs []string)

	// iptables
	appendRule() error
	releaseRule() error

	// export DBus service
	export(service *dbusutil.Service) error
}

//func CreateProxyService() error {
//	// get session bus
//	service, err := dbusutil.NewSessionService()
//	if err != nil {
//		logger.Warningf("get session bus failed, err: %v", err)
//		return err
//	}
//
//	// export app proxy
//	app := newProxy(tProxy.AppProxy)
//	if app == nil {
//		logger.Warning("app proxy init failed")
//		return errors.New("app proxy init failed")
//	}
//	err = app.export(service)
//	if err != nil {
//		return err
//	}
//
//	// export global proxy
//	global := newProxy(tProxy.GlobalProxy)
//	if global == nil {
//		logger.Warning("global proxy init failed")
//		return errors.New("global proxy init failed")
//	}
//	err = global.export(service)
//	if err != nil {
//		return err
//	}
//
//	// request name
//	err = service.RequestName(BusServiceName)
//	if err != nil {
//		logger.Warningf("request service failed, err: %v", err)
//		return err
//	}
//
//	logger.Debug("success export DBus service")
//
//	service.Wait()
//	return nil
//}

// new proxy according to scope
func newProxy(scope define.Scope) BaseProxy {
	switch scope {
	case define.App:
		return NewAppProxy()
	case define.Global:
		return NewGlobalProxy()
	default:
		logger.Warningf("init unknown scope type")
		return nil
	}
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
