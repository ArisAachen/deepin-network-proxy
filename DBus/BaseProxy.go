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
	StopProxy() *dbus.Error
	SetProxies(proxies config.ScopeProxies) *dbus.Error
	ClearProxies() *dbus.Error

	// manager
	loadConfig()
	saveManager(manager *Manager)

	// getScope() tProxy.ProxyScope
	getDBusPath() dbus.ObjectPath
	getScope() define.Scope

	// get cgroup v2 level
	getCGroupPriority() define.Priority

	//// cgroup v2
	//addCGroupExes(procs []string)
	//delCGroupExes(procs []string)

	// iptables
	appendRule() error
	releaseRule() error

	// export DBus service
	export(service *dbusutil.Service) error
}

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
