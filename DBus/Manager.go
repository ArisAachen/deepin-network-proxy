package DBus

import (
	"sync"

	define "github.com/DeepinProxy/Define"
	newCGroups "github.com/DeepinProxy/NewCGroups"
	newIptables "github.com/DeepinProxy/NewIptables"
	netlink "github.com/linuxdeepin/go-dbus-factory/com.deepin.system.procs"
	"pkg.deepin.io/lib/dbusutil"
)

// manage all proxy handler
type Manager struct {

	// dbus
	procsService *netlink.Procs
	sesService   *dbusutil.Service
	sysService   *dbusutil.Service
	sigLoop      *dbusutil.SignalLoop

	// proxy handler
	handler []*BaseProxy

	// cgroup manager
	mainController *newCGroups.Controller
	controllerMgr  *newCGroups.Manager

	// iptables manager
	mainChain   *newIptables.Chain // main attach chain
	iptablesMgr *newIptables.Manager

	// if current listening
	runOnce *sync.Once
}

// inti manager
func (m *Manager) Init() error {
	// init session dbus service to export service
	sesService, err := dbusutil.NewSessionService()
	if err != nil {
		logger.Warningf("init dbus session service failed, err:  %v", err)
		return err
	}
	m.sesService = sesService

	// init system dbus service to monitor service
	service, err := dbusutil.NewSystemService()
	if err != nil {
		logger.Warningf("init dbus system service failed, err:  %v", err)
		return err
	}
	// store service
	m.sysService = service
	// attach dbus object
	m.procsService = netlink.NewProcs(service.Conn())
	m.sigLoop = dbusutil.NewSignalLoop(service.Conn(), 10)
	m.controllerMgr = newCGroups.NewManager()
	return nil
}

// create handler and export service
func (m *Manager) Export() error {
	// app
	appProxy := NewAppProxy()
	// create cgroups controller
	appController, err := m.controllerMgr.CreatePriorityController(appProxy.scope.String(), appProxy.getCGroupLevel())
	if err != nil {
		logger.Warningf("create app proxy controller failed, err: %v", err)
		return err
	}
	// set controller
	appProxy.setController(appController)
	// export app dbus path
	err = m.sesService.Export(appProxy.getDBusPath(), appProxy)
	if err != nil {
		logger.Warningf("export app proxy failed, err: %v", err)
		return err
	}
	// m.handler = append(m.handler,appProxy)

	// global
	globalProxy := NewGlobalProxy()
	globalController, err := m.controllerMgr.CreatePriorityController(globalProxy.scope.String(), globalProxy.getCGroupLevel())
	if err != nil {
		logger.Warningf("create global proxy controller failed, err: %v", err)
		return err
	}
	// set controller
	globalProxy.setController(globalController)
	err = m.sesService.Export(globalProxy.getDBusPath(), globalProxy)
	if err != nil {
		logger.Warningf("export app proxy failed, err: %v", err)
		return err
	}

	// request dbus service
	err = m.sesService.RequestName(BusServiceName)
	if err != nil {
		logger.Warningf("request service name failed, err: %v", err)
		return err
	}
	return nil
}

// only run once method
func (m *Manager) Start() {
	// if need reset once
	if m.runOnce == nil {
		m.runOnce = new(sync.Once)
	}
	m.runOnce.Do(func() {
		// init cgroups
		_ = m.initCGroups()
		// iptables init
		_ = m.initIptables()
	})
}

// init iptables
func (m *Manager) initIptables() error {
	var err error
	m.iptablesMgr = newIptables.NewManager()
	m.iptablesMgr.Init()
	// get mangle output chain
	outputChain := m.iptablesMgr.GetChain("mangle", "OUTPUT")
	// create main chain to manager all children chain
	// sudo iptables -t mangle -N Main
	// sudo iptables -t mangle -A OUTPUT -j main
	m.mainChain, err = outputChain.CreateChild(define.Main, 0, &newIptables.CompleteRule{Action: define.Main})

	// mainChain add default rule
	// iptables -t mangle -A All_Entry -m cgroup --path main.slice -j RETURN
	extends := newIptables.ExtendsRule{
		// -m
		Match: "m",
		// cgroup --path main.slice
		Elem: newIptables.ExtendsElem{
			// cgroup
			Match: "cgroup",
			// --path main.slice
			Base: newIptables.BaseRule{Match: "path", Param: define.Main + ".slice"},
		},
	}
	// one complete rule
	cpl := &newIptables.CompleteRule{
		// -j RETURN
		Action: newIptables.RETURN,
		BaseSl: nil,
		// -m cgroup --path main.slice -j RETURN
		ExtendsSl: []newIptables.ExtendsRule{extends},
	}
	// append rule
	err = m.mainChain.AppendRule(cpl)
	return err
}

// init cgroups
func (m *Manager) initCGroups() error {
	// create controller
	var err error
	m.mainController, err = m.controllerMgr.CreatePriorityController(define.Main, define.MainPriority)
	if err != nil {
		return err
	}


	return nil
}

// start listen
func (m *Manager) Listen() error {
	m.procsService.InitSignalExt(m.sigLoop, true)
	_, err := m.procsService.ConnectExecProc(func(execPath string, cwdPath string, pid string) {
		// search controller according to exe path, get highest priority one
		controller := m.controllerMgr.GetControllerByCtlPath(execPath)
		proc := &netlink.ProcMessage{
			ExecPath: execPath,
			Pid:      pid,
		}
		// add to cgroups.procs and save
		err := controller.AddCtrlProc(proc)
		if err != nil {
			logger.Warningf("[%s] add exec %s to cgroups failed, err: %v", controller.Name, execPath, err)
		}
	})
	if err != nil {
		logger.Warningf("connect exec proc failed, err: %v")
		return err
	}
	_, err = m.procsService.ConnectExitProc(func(execPath string, cwdPath string, pid string) {
		// search controller according to exe path
		controller := m.controllerMgr.GetControllerByCtlPath(execPath)
		proc := &netlink.ProcMessage{
			ExecPath: execPath,
			Pid:      pid,
		}
		// del from save
		err := controller.DelCtlProc(proc)
		if err != nil {
			logger.Warningf("[%s] del exec %s from cgroups failed, err: %v", controller.Name, execPath, err)
		}
	})
	if err != nil {
		logger.Warningf("connect exit proc failed, err: %v")
		return err
	}
	m.sigLoop.Start()
	return nil
}

// release all source
func (m *Manager) Release() error {
	// check if all app and global proxy has stopped
	if m.mainChain.GetChildrenCount() != 0 {
		return nil
	}
	// remove all handler
	m.procsService.RemoveAllHandlers()
	// stop loop
	m.sigLoop.Stop()

	// remove chain
	err := m.mainChain.Remove()
	if err != nil {
		logger.Warningf("remove main chain failed, err: %v", err)
		return err
	}

	// reset once
	m.runOnce = nil
	return nil
}
