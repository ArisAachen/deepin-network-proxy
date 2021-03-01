package DBus

import (
	"fmt"
	cgroup "github.com/DeepinProxy/CGroups"
	config "github.com/DeepinProxy/Config"
	iptables "github.com/DeepinProxy/Iptables"
	tProxy "github.com/DeepinProxy/TProxy"
	"github.com/godbus/dbus"
	"pkg.deepin.io/lib/dbusutil"
	"strconv"
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
		proxyPrv: initProxyPrv(tProxy.AppProxy),
	}
	// load config
	appModule.loadConfig()

	// init cgroup
	_ = appModule.initCGroup()
	apps := appModule.Proxies.ProxyProgram
	appModule.addCGroupExes(apps)

	_ = appModule.initIptables()

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

// rewrite get scope
func (mgr *AppProxy) getScope() tProxy.ProxyScope {
	return tProxy.AppProxy
}

func (mgr *AppProxy) getCGroupLevel() int {
	return cgroup.AppProxyLevel
}

// rewrite export DBus path
func (mgr *AppProxy) getDBusPath() dbus.ObjectPath {
	path := BusPath + "/" + tProxy.AppProxy.String()
	return dbus.ObjectPath(path)
}

// init cgroup
func (mgr *AppProxy) initCGroup() error {
	// will not error in any case
	err := mgr.proxyPrv.initCGroup()
	if err != nil {
		return err
	}
	// make dir
	member, err := allCGroups.CreateCGroup(cgroup.AppProxyLevel, mgr.scope.String())
	if err != nil {
		return err
	}
	mgr.cgroupMember = member
	logger.Debugf("[%s] create cgroup success", mgr.scope)
	return nil
}

func (mgr *AppProxy) initIptables() error {
	// init global iptables
	err := mgr.proxyPrv.initIptables()
	if err != nil {
		logger.Warningf("[%s] init global iptables failed, err: %v", err)
		return err
	}
	extends := []iptables.ExtendsRule{
		iptables.ExtendsRule{
			Match: "m",
			Base: []iptables.ExtendsElem{
				{
					Match: "cgroup",
					Base: []iptables.BaseRule{
						{
							Match: "path",
							Param: mgr.cgroupMember.GetCGroupPath(),
						},
					},
				},
			},
		},
	}
	err = allIptables.CreateChain("mangle", "OUTPUT", cgroup.AppProxyLevel-1, "App", nil, extends)
	if err != nil {
		logger.Warningf("[%s] create chain failed, err: %v", err)
		return err
	}
	// -t mangle -I OUTPUT -p tcp -j MARK -j MARK --set-mark 8090
	bs := []iptables.BaseRule{
		{
			Match: "-set-mark",
			Param: strconv.Itoa(mgr.Proxies.TPort),
		},
		{
			Match: "p",
			Param: "tcp",
		},
	}
	// add mark
	err = allIptables.AddRule("mangle", "App", iptables.MARK, bs, nil)
	if err != nil {
		logger.Warningf("[%s] add OUTPUT rule failed, err: %v", mgr.scope, err)
		return err
	}
	logger.Debugf("[%s] add mark success", mgr.scope)

	extends = []iptables.ExtendsRule{
		iptables.ExtendsRule{
			Match: "m",
			Base: []iptables.ExtendsElem{
				{
					Match: "mark",
					Base: []iptables.BaseRule{
						{
							Match: "mark",
							Param: strconv.Itoa(mgr.Proxies.TPort),
						},
					},
				},
			},
		},
	}

	bs = []iptables.BaseRule{
		{
			Match: "p",
			Param: "tcp",
		},
		{
			Match: "-on-port",
			Param: strconv.Itoa(mgr.Proxies.TPort),
		},
	}
	// transparent mark
	err = allIptables.AddRule("mangle", "PREROUTING", iptables.TPROXY, bs, extends)
	if err != nil {
		logger.Warningf("[%s] add transparent rule failed, err: %v", mgr.scope, err)
	}
	return nil
}

// add proxy app
func (mgr *AppProxy) AddProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.addCGroupExes(apps)
	return nil
}

// delete proxy app
func (mgr *AppProxy) DelProxyApps(apps []string) *dbus.Error {
	mgr.proxyPrv.delCGroupExes(apps)
	return nil
}
