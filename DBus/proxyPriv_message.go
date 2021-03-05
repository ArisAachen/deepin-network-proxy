package DBus

import (
	define "github.com/DeepinProxy/Define"
	"github.com/godbus/dbus"
)

// scope
// rewrite get scope
func (mgr *proxyPrv) getScope() define.Scope {
	return mgr.scope
}

func (mgr *proxyPrv) getDBusPath() dbus.ObjectPath {
	path := BusPath + "/" + mgr.scope.ToString()
	return dbus.ObjectPath(path)
}
