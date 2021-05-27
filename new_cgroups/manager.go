package NewCGroups

import (
	"errors"
	"sort"

	com "github.com/ArisAachen/deepin-network-proxy/com"
	define "github.com/ArisAachen/deepin-network-proxy/define"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

type Manager struct {
	controllers []*Controller
}

// create manager
func NewManager() *Manager {
	manager := &Manager{
		controllers: []*Controller{},
	}
	return manager
}

// create controller handler
func (m *Manager) CreatePriorityController(name define.Scope, uid int, gid int, priority define.Priority) (*Controller, error) {
	if m.CheckControllerExist(name, priority) {
		return nil, errors.New("controller name or priority already exist")
	}
	// create controller
	controller := &Controller{
		Name:       name,
		Priority:   priority,
		manager:    m,
		CtlPathSl:  []string{},
		CtlProcMap: make(map[string]ControlProcSl),
	}
	// make dir
	err := com.GuaranteeDir(controller.GetControlPath())
	if err != nil {
		return nil, err
	}
	//err = os.Chown(controller.GetCGroupPath(), uid, gid)
	//if err != nil {
	//	return nil, err
	//}
	// append controller
	m.controllers = append(m.controllers, controller)
	sort.SliceStable(m.controllers, func(i, j int) bool {
		// sort by priority
		if m.controllers[i].Priority > m.controllers[j].Priority {
			return false
		}
		return true
	})
	return controller, nil
}

// get controller by control app path
func (m *Manager) GetControllerByCtlPath(path string) *Controller {
	// search app name
	for _, controller := range m.controllers {
		if controller.CheckCtlPathSl(path) {
			logger.Debugf("[%s] controller find app path %s", controller.Name, path)
			return controller
		}
	}
	logger.Debugf("app path %s cant found in any controller", path)
	return nil
}

// get controller by control pid
func (m *Manager) GetControllerByCtrlByPPid(ppid string) *Controller {
	// search ppid
	for _, controller := range m.controllers {
		if controller.CheckCtrlPid(ppid) != nil {
			logger.Debugf("[%s] controller find ppid  %s", controller.Name, ppid)
			return controller
		}
	}
	logger.Debugf("ppid %s cant found in any controller ", ppid)
	return nil
}

// check if name controller already exist
func (m *Manager) CheckControllerExist(name define.Scope, priority define.Priority) bool {
	// search name
	for _, controller := range m.controllers {
		if controller.Name == name || controller.Priority == priority {
			return true
		}
	}
	return false
}

// get controller count
func (m *Manager) GetControllerCount() int {
	return len(m.controllers)
}

// init
func init() {
	logger = log.NewLogger("daemon/cgroup")
	logger.SetLogLevel(log.LevelInfo)
}
