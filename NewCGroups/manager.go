package NewCGroups

import (
	"errors"

	com "github.com/DeepinProxy/Com"
	netlink "github.com/linuxdeepin/go-dbus-factory/com.deepin.system.procs"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

type Manager struct {
	controllers []*Controller
}

// get controller index according to name
func (m *Manager) GetCreateControllerIndex(name string) (int, bool) {
	for index, controller := range m.controllers {
		if controller.Name == name {
			return index, true
		}
	}
	return 0, false
}

// check if index is valid
func (m *Manager) validIndex(index int) bool {
	return len(m.controllers) >= index
}

// create controller under cgroup/unified/
func (m *Manager) CreateAppendController(name string) (*Controller, error) {
	// check if controller exist
	if _, exist := m.GetCreateControllerIndex(name); exist {
		return nil, errors.New("controller already exist")
	}
	// create controller
	controller := &Controller{
		Name:       name,
		manager:    m,
		CtlPathSl:  []string{},
		CtlProcMap: make(map[string][]*netlink.ProcMessage),
	}
	// append controller
	m.controllers = append(m.controllers, controller)
	return controller, nil
}

// create controller under cgroup/unified/
func (m *Manager) CreateAfterController(name string, index int) (*Controller, error) {
	// check index
	if m.validIndex(index) {
		return nil, errors.New("index is invalid")
	}
	// check if already exist
	if _, exist := m.GetCreateControllerIndex(name); exist {
		return nil, errors.New("controller already exist")
	}
	controller := &Controller{
		Name:       name,
		manager:    m,
		CtlPathSl:  []string{},
		CtlProcMap: make(map[string][]*netlink.ProcMessage),
	}
	// insert index
	ifc, update, err := com.MegaInsert(m.controllers, controller, index)
	if err != nil || update {
		return nil, err
	}
	// convert
	temp, ok := ifc.([]*Controller)
	if !ok {
		return nil, errors.New("convert failed")
	}
	m.controllers = temp
	return controller, nil
}

// init
func init() {
	logger = log.NewLogger("daemon/cgroup")
	logger.SetLogLevel(log.LevelDebug)
}
