package DBus

import (
	com "github.com/ArisAachen/deepin-network-proxy/com"
	"os"
	"path/filepath"
	"pkg.deepin.io/lib/log"
	"sync"

	config "github.com/ArisAachen/deepin-network-proxy/config"
	define "github.com/ArisAachen/deepin-network-proxy/define"
	route "github.com/ArisAachen/deepin-network-proxy/ip_route"
	newCGroups "github.com/ArisAachen/deepin-network-proxy/new_cgroups"
	newIptables "github.com/ArisAachen/deepin-network-proxy/new_iptables"
	"pkg.deepin.io/lib/dbusutil"
)

// manage all proxy handler
type Manager struct {

	// dbus
	// procsService netlink.Procs
	sesService *dbusutil.Service
	sysService *dbusutil.Service
	// sigLoop      *dbusutil.SignalLoop

	// proxy handler
	handler []BaseProxy

	// cgroup manager
	mainController *newCGroups.Controller
	controllerMgr  *newCGroups.Manager

	// config
	config *config.ProxyConfig

	// iptables manager
	mainChain   *newIptables.Chain // main attach chain
	iptablesMgr *newIptables.Manager

	// route manager
	mainRoute *route.Route
	routeMgr  *route.Manager

	// if current listening
	runOnce *sync.Once
}

// make manager
func NewManager() *Manager {
	manager := &Manager{

	}
	return manager
}

// inti manager
func (m *Manager) Init() error {
	// init session dbus service to export service
	sysService, err := dbusutil.NewSystemService()
	if err != nil {
		logger.Warningf("init dbus session service failed, err:  %v", err)
		return err
	}
	// store service
	m.sysService = sysService
	// attach dbus objects
	// m.procsService = netlink.NewProcs(sysService.Conn())
	// m.sigLoop = dbusutil.NewSignalLoop(sysService.Conn(), 10)
	return nil
}

// load config
func (m *Manager) LoadConfig() error {
	// get effective user config dir
	path, err := com.GetConfigDir()
	if err != nil {
		logger.Warningf("failed to get user home dir, user:%v, err: %v", os.Geteuid(), err)
		return err
	}
	path = filepath.Join(path, define.ConfigName)
	// config
	m.config = config.NewProxyCfg()
	err = m.config.LoadPxyCfg(path)
	if err != nil {
		logger.Warningf("load config failed, path: %s, err: %v", path, err)
		return err
	}
	return nil
}

// write config
func (m *Manager) WriteConfig() error {
	// get config path
	path, err := com.GetConfigDir()
	if err != nil {
		logger.Warningf("[manager] get user home dir failed, user:%v, err: %v", os.Geteuid(), err)
		return err
	}
	path = filepath.Join(path, define.ConfigName)
	err = m.config.WritePxyCfg(path)
	if err != nil {
		logger.Warningf("[manager] write config file failed, err: %v", err)
		return err
	}
	return nil
}

// create handler and export service
func (m *Manager) Export() error {
	// app
	appProxy := newProxy(define.App)
	// save manager
	appProxy.saveManager(m)
	// load config
	appProxy.loadConfig()
	// export
	err := appProxy.export(m.sysService)
	if err != nil {
		logger.Warningf("create app proxy controller failed, err: %v", err)
		return err
	}
	m.handler = append(m.handler, appProxy)

	//// global
	//globalProxy := newProxy(define.Global)
	//// save manager
	//globalProxy.saveManager(m)
	//// load config
	//globalProxy.loadConfig()
	//// export
	//err = globalProxy.export(m.sysService)
	//if err != nil {
	//	logger.Warningf("export app proxy failed, err: %v", err)
	//	return err
	//}
	// m.handler = append(m.handler, globalProxy)

	// request dbus service
	err = m.sysService.RequestName(BusServiceName)
	if err != nil {
		logger.Warningf("request service name failed, err: %v", err)
		return err
	}
	return nil
}

func (m *Manager) Wait() {
	m.sysService.Wait()
}

// only run once method
func (m *Manager) Start() {
	// if need reset once
	if m.runOnce == nil {
		m.runOnce = new(sync.Once)
	}
	m.runOnce.Do(func() {
		// run first clean script
		_ = m.firstClean()

		// init cgroups
		_ = m.initCGroups()

		// iptables init
		_ = m.initIptables()

		// init route
		_ = m.initRoute()
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
	// sudo iptables -t mangle -A OUTPUT -j Main
	m.mainChain, err = outputChain.CreateChild(define.Main.String(), 0, &newIptables.CompleteRule{Action: define.Main.String()})
	if err != nil {
		logger.Warningf("init iptables failed, err: %v", err)
		return err
	}
	// dont proxy local lo
	// sudo iptables -t mangle -A Main 1 -o lo -j RETURN
	base := newIptables.BaseRule{
		// -o
		Match: "o",
		// "lo"
		Param: "lo",
	}
	cpl := &newIptables.CompleteRule{
		Action:    newIptables.RETURN,
		BaseSl:    []newIptables.BaseRule{base},
		ExtendsSl: nil,
	}
	// append rule
	err = m.mainChain.AppendRule(cpl)
	if err != nil {
		logger.Warningf("init iptables failed, err: %v", err)
		return err
	}

	// mainChain add default rule
	// iptables -t mangle -A Main -m cgroup --path main.slice -j RETURN
	extends := newIptables.ExtendsRule{
		// -m
		Match: "m",
		// cgroup --path main.slice
		Elem: newIptables.ExtendsElem{
			// cgroup
			Match: "cgroup",
			// --path main.slice
			Base: newIptables.BaseRule{
				Match: "path", Param: define.Main.String() + ".slice",
			},
		},
	}
	// one complete rule
	cpl = &newIptables.CompleteRule{
		// -j RETURN
		Action: newIptables.RETURN,
		BaseSl: nil,
		// -m cgroup --path main.slice -j RETURN
		ExtendsSl: []newIptables.ExtendsRule{extends},
	}
	// append rule
	err = m.mainChain.AppendRule(cpl)
	if err != nil {
		logger.Warningf("init iptables failed, err: %v", err)
		return err
	}
	logger.Debug("init iptables success")
	return err
}

// init cgroups
func (m *Manager) initCGroups() error {
	m.controllerMgr = newCGroups.NewManager()
	// create controller
	var err error
	m.mainController, err = m.controllerMgr.CreatePriorityController(define.Main, 0, define.MainPriority)
	if err != nil {
		logger.Warningf("init cgroup failed, err: %v", err)
		return err
	}
	logger.Debug("init cgroup success")
	return nil
}

// init route
func (m *Manager) initRoute() error {
	var err error
	m.routeMgr = route.NewManager()
	node := route.RouteNodeSpec{
		Type:   "local",
		Prefix: "default",
	}
	info := route.RouteInfoSpec{
		Dev: "lo",
	}
	m.mainRoute, err = m.routeMgr.CreateRoute("100", node, info)
	if err != nil {
		logger.Warningf("init route failed, err: %v", err)
		return err
	}
	logger.Debug("init route success")
	return nil
}

// format current procs
func (m *Manager) GetAllProcs() (map[string]newCGroups.ControlProcSl, error) {
	// check service
	//if m.procsService == nil {
	//	logger.Warning("[manager] get procs failed, service not init")
	//	return nil, errors.New("service not init")
	//}
	// get procs message
	// map[pid]{pid exec cgroups}
	//procs, err := m.procsService.Procs().Get(0)
	//if err != nil {
	//	logger.Warningf("[%s] get procs failed, err: %v", "manager", err)
	//	return nil, err
	//}
	// map[exec][pid exec cgroups]
	//ctrlProcMap := make(map[string]newCGroups.ControlProcSl)
	//for _, proc := range procs {
	//	execPath := proc.ExecPath
	//	ctrlProcSl, ok := ctrlProcMap[execPath]
	//	// if not exist, add one
	//	if !ok {
	//		ctrlProcSl = newCGroups.ControlProcSl{}
	//		// ctrlProcMap[execPath] = ctrlProcSl
	//	}
	//	// append
	//	var temp netlink.ProcMessage = proc
	//	ctrlProcSl = append(ctrlProcSl, &temp)
	//	ctrlProcMap[execPath] = ctrlProcSl
	//}
	return nil, nil
}

// start listen
func (m *Manager) Listen() error {
	//m.sigLoop.Start()
	//m.procsService.InitSignalExt(m.sigLoop, true)
	//_, err := m.procsService.ConnectExecProc(func(execPath string, cgroupPath string, pid string, ppid string) {
	//	proc := &netlink.ProcMessage{
	//		ExecPath:   execPath,
	//		CGroupPath: cgroupPath,
	//		Pid:        pid,
	//		PPid:       ppid,
	//	}
	//	logger.Debugf("listen exec proc %v", proc)
	//	// check if is child proc
	//	controller := m.controllerMgr.GetControllerByCtrlByPPid(ppid)
	//	if controller != nil {
	//		// cover proc
	//		parent := controller.CheckCtrlPid(ppid)
	//		proc.ExecPath = parent.ExecPath
	//		proc.CGroupPath = parent.CGroupPath
	//		// add to
	//		err := controller.AddCtrlProc(proc)
	//		if err != nil {
	//			logger.Warningf("[%s] add exec %s to cgroups failed, err: %v", controller.Name, execPath, err)
	//		}
	//		return
	//	}
	//
	//	// search controller according to exe path, get highest priority one
	//	controller = m.controllerMgr.GetControllerByCtlPath(execPath)
	//	if controller == nil {
	//		return
	//	}
	//	logger.Infof("start proc %s need add to proxy", execPath)
	//	// add to cgroups.procs and save
	//	err := controller.AddCtrlProc(proc)
	//	if err != nil {
	//		logger.Warningf("[%s] add exec %s to cgroups failed, err: %v", controller.Name, execPath, err)
	//	}
	//
	//})
	//if err != nil {
	//	logger.Warningf("connect exec proc failed, err: %v")
	//	return err
	//}
	//_, err = m.procsService.ConnectExitProc(func(execPath string, cgroupPath string, pid string, ppid string) {
	//	// search controller according to exe path
	//	logger.Debugf("listen exit proc %v", execPath)
	//	controller := m.controllerMgr.GetControllerByCtlPath(execPath)
	//	if controller == nil {
	//		return
	//	}
	//	proc := &netlink.ProcMessage{
	//		ExecPath:   execPath,
	//		CGroupPath: cgroupPath,
	//		Pid:        pid,
	//		PPid:       ppid,
	//	}
	//	logger.Infof("start proc %s need remove from proxy", execPath)
	//	// del from save
	//	err := controller.DelCtlProc(proc)
	//	if err != nil {
	//		logger.Warningf("[%s] del exec %s from cgroups failed, err: %v", controller.Name, execPath, err)
	//	}
	//
	//})
	return nil
}

// release all source
func (m *Manager) release() error {
	// check if all app and global proxy has stopped
	if m.mainChain.GetChildrenCount() != 0 {
		return nil
	}
	// remove all handler
	// m.procsService.RemoveAllHandlers()
	// stop loop
	// m.sigLoop.Stop()

	// remove chain
	err := m.mainChain.Remove()
	if err != nil {
		logger.Warningf("[manager] remove main chain failed, err: %v", err)
		return err
	}
	m.iptablesMgr = nil

	//// release all control procs
	//err = m.mainController.ReleaseAll()
	//if err != nil {
	//	logger.Warning("[manager] release all control procs failed, err:", err)
	//	return err
	//}
	//m.controllerMgr = nil

	// remove all route
	err = m.mainRoute.Remove()
	if err != nil {
		logger.Warning("[manager] remove all route failed, err:", err)
		return err
	}
	m.routeMgr = nil

	// reset once
	m.runOnce = nil
	return nil
}

// run first clean script
func (m *Manager) firstClean() error {
	// get config path
	path, err := com.GetConfigDir()
	if err != nil {
		logger.Warningf("[%s] run first clean failed, config err: %v", "manager", err)
		return err
	}
	// get script file path
	path = filepath.Join(path, define.ScriptName)
	// run script
	buf, err := com.RunScript(path, []string{"clear_Main"})
	if err != nil {
		logger.Debugf("[%s] run first clean script failed, out: %s, err: %v", "manager", string(buf), err)
		return err
	}
	logger.Debugf("[%s] run first clean script success", "manager")
	return nil
}

// first adjust cgroups
func (m *Manager) firstAdjustCGroups() error {
	// get all procs message
	procsMap, err := m.GetAllProcs()
	if err != nil {
		return err
	}

	// add map
	for path, procSl := range procsMap {
		// check if exist
		if !com.MegaExist(mainProxy, path) {
			logger.Debugf("[%s] dont need add %s at first", "manager", path)
			continue
		}

		// check if already exist
		controller := m.controllerMgr.GetControllerByCtlPath(path)
		if controller == nil {
			// add path
			m.mainController.AddCtlAppPath(path)
			err := m.mainController.MoveIn(path, procSl)
			if err != nil {
				logger.Warning("[%s] add procs %s at first failed, err: %v", "manager", path, err)
				continue
			}
			logger.Debugf("[%s] add procs %s at first success", "manager", path)
		} else {
			err = m.mainController.UpdateFromManager(path)
			if err != nil {
				logger.Warning("[%s] add proc %s from %s at first failed, err: %v", "manager", path, controller.Name, err)
			} else {
				logger.Debugf("[%s] add proc %s from %s at first failed", "manager", path, controller.Name)
			}
			m.mainController.AddCtlAppPath(path)
		}
	}
	return nil
}

func init() {
	logger = log.NewLogger("daemon/iptables")
	logger.SetLogLevel(log.LevelDebug)
}
