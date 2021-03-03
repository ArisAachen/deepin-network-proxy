package DBus

import (
	"fmt"
	cgroup "github.com/DeepinProxy/CGroups"
	config "github.com/DeepinProxy/Config"
	newIptables "github.com/DeepinProxy/NewIptables"
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

	// init iptables
	appModule.createTable()

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

// init new iptables
func (mgr *AppProxy) createTable() {
	err := mgr.proxyPrv.initNewIptables()
	if err != nil {
		return
	}
	mainChain := allNewIptables.GetChain("mangle", "MainEntry")
	if mainChain == nil {
		logger.Warning("main chain has no entry")
		return
	}
	// check if global exist
	index, exist := mainChain.GetCreateChildIndex("GlobalEntry")
	if !exist {
		// correct index if not exist
		index = mainChain.GetRulesCount()
	}
	// command line
	// iptables -t mangle -I All_Entry $1 -p tcp -m cgroup --path app.slice -j App_Proxy
	cpl := &newIptables.CompleteRule{
		// base rules slice         -p tcp
		BaseSl: []newIptables.BaseRule{
			{
				Match: "p",
				Param: "tcp",
			},
		},
		// extends rules slice       -m cgroup --path app.slice -j App_Proxy
		ExtendsSl: []newIptables.ExtendsRule{
			{
				Match: "m",
				Elem: newIptables.ExtendsElem{
					Match: "cgroup",
					Base: newIptables.BaseRule{
						Match: "path",
						Param: mgr.scope.String(),
					},
				},
			},
		},
	}
	childChain, err := mainChain.CreateChild("App", index, cpl)
	if err != nil {
		return
	}
	mgr.chains[1] = childChain
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
