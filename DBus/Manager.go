package DBus

import (
	newCGroups "github.com/DeepinProxy/NewCGroups"
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
	controllerMgr *newCGroups.Manager
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

// start listen
func (m *Manager) Listen() error {
	m.procsService.InitSignalExt(m.sigLoop, true)
	_, err := m.procsService.ConnectExecProc(func(execPath string, cwdPath string, pid string) {
		// search controller according to exe path
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
		// add to cgroups.procs and save
		err := controller.DelCtlProc(proc, true)
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
func (m *Manager) Release() {
	// remove all handler
	m.procsService.RemoveAllHandlers()
	// stop loop
	m.sigLoop.Stop()
}
