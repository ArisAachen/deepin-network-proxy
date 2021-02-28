package CGroups

import (
	"errors"
	"fmt"
	"github.com/godbus/dbus"
	"pkg.deepin.io/lib/log"
	"sort"
	"sync"

	com "github.com/DeepinProxy/Com"
	netlink "github.com/linuxdeepin/go-dbus-factory/com.deepin.system.procs"
)

var logger *log.Logger

// cgroup v2 cgroup.tgtExeSl

// cgroup v2 /sys/fs/cgroup/unified
type CGroupManager struct {
	CGroups []*CGroupMember // map[priority]CGroupMember

	// proc service
	procsService *netlink.Procs

	// all procs message
	// procMap map[string][]ProcMessage

	// lock
	lock sync.Mutex
}

// create CGroupManager
func NewCGroupManager() *CGroupManager {
	return &CGroupManager{
		CGroups: []*CGroupMember{},
		// procMap: make(map[string][]ProcMessage),
	}
}

// create cgroup path
func (c *CGroupManager) CreateCGroup(level int, elemPath string) (*CGroupMember, error) {
	// lock
	c.lock.Lock()
	defer c.lock.Unlock()
	// check if cgroup already exist
	for _, cgroup := range c.CGroups {
		if cgroup.path == elemPath {
			logger.Warningf("create group failed, path [%s] already exist", elemPath)
			return nil, fmt.Errorf("create group failed, path [%s] already exist", elemPath)
		}
		if cgroup.priority == level {
			logger.Warningf("create group failed, level [%d] already exist", level)
			return nil, fmt.Errorf("create group failed, level [%d] already exist", level)
		}
	}
	// use level to mark priority
	member := &CGroupMember{
		parent:   c,
		path:     elemPath,
		priority: level,
		tgtExeSl: []string{},
		procMap:  make(map[string][]ProcMessage),
	}
	// add to manager
	c.CGroups = append(c.CGroups, member)
	// sort slice
	sort.SliceStable(c.CGroups, func(i, j int) bool {
		// check if priority is sorted correctly
		if c.CGroups[i].priority > c.CGroups[j].priority {
			return false
		}
		return true
	})

	// make dir
	cgpProcs := member.getProcsPath()
	err := com.GuaranteeDir(cgpProcs)
	if err != nil {
		logger.Warningf("mkdir [%s] failed, err: %v", cgpProcs, err)
		return nil, err
	}
	return member, nil
}

//// add proc path to cgroup elem
//func (c *CGroupManager) AddCGroupProcs(elem string, procs []string) {
//	// lock
//	c.lock.Lock()
//	defer c.lock.Unlock()
//
//	// add
//	for _, cgroup := range c.CGroups {
//		if cgroup.path == elem {
//			logger.Debugf("cgroup [%s] found in manager, begin to add tgtExeSl [%s]", elem, procs)
//			cgroup.AddTgtExes(procs)
//			return
//		}
//	}
//	// if not found, out put log
//	logger.Warningf("cgroup [%s] not found in manager", elem)
//}
//
//// add proc path to cgroup elem
//func (c *CGroupManager) DelCGroupProcs(elem string, procs []string) {
//	// lock
//	c.lock.Lock()
//	defer c.lock.Unlock()
//
//	// add
//	for _, cgroup := range c.CGroups {
//		if cgroup.path == elem {
//			logger.Debugf("cgroup [%s] found in manager, begin to del proc [%s]", elem, procs)
//			cgroup.delTgtExes(procs)
//			return
//		}
//	}
//	// if not found, out put log
//	logger.Warningf("cgroup [%s] not found in manager", elem)
//}

func (c *CGroupManager) GetCGroupMember(exe string) *CGroupMember {
	// lock
	c.lock.Lock()
	defer c.lock.Unlock()

	// search which cgroup, proc exist
	for _, cgroup := range c.CGroups {
		if cgroup.existTgtExe(exe) {
			logger.Debugf("exe [%s] is found in [%s]", exe, cgroup.path)
			return cgroup
		}
	}
	logger.Debugf("exe [%s] cant found in any cgroup", exe)
	return nil
}

func (c *CGroupManager) GetCGroupProcsPath(exe string) string {
	// search which cgroup, proc exist
	cgroup := c.GetCGroupMember(exe)
	if cgroup == nil {
		return ""
	}
	// if found
	return cgroup.path
}

func (c *CGroupManager) getProcs() (map[string][]ProcMessage, error) {
	// check if proc service is nil
	if c.procsService == nil {
		logger.Warning("get proc failed, proc service nit init")
		return nil, errors.New("get proc failed, proc service nit init")
	}
	// get procs from DBus service
	procs, err := c.procsService.Procs().Get(0)
	if err != nil {
		logger.Warningf("get procs failed, err: %v", err)
		return nil, err
	}
	// value map[string]ProcMessage

	// temp proc message
	temProcs := make(map[string][]ProcMessage)
	for _, proc := range procs {
		// get slice
		procSl, ok := temProcs[proc.ExecPath]
		if !ok {
			procSl = []ProcMessage{}
			temProcs[proc.ExecPath] = procSl
		}
		// mega add
		ifc, _, err := com.MegaAdd(procSl, proc)
		if err != nil {
			logger.Warningf("mega add failed, err: %v", err)
			continue
		}
		// convert slice
		procSl, ok = ifc.([]ProcMessage)
		if !ok {
			logger.Warning("convert ProcMessage slice failed")
			continue
		}
		temProcs[proc.ExecPath] = procSl
	}
	logger.Debugf("get procs success, procs: %v", temProcs)
	return temProcs, nil
}

func (c *CGroupManager) Listen() error {
	systemBus, err := dbus.SystemBus()
	if err != nil {
		logger.Warningf("create system bus failed, err: %v", err)
		return err
	}
	c.procsService = netlink.NewProcs(systemBus)
	// listen proc exec
	_, err = c.procsService.ConnectExecProc(func(execPath string, cgroup2Path string, pid string) {
		// get cgroup member
		cgroup := c.GetCGroupMember(execPath)
		if cgroup == nil {
			logger.Debugf("exe [%s] cant found in any cgroup", execPath)
			return
		}
		// make message
		proc := ProcMessage{
			execPath:    execPath,
			pid:         pid,
			cgroup2Path: cgroup2Path,
		}
		// add proc to cgroup
		err = cgroup.addCrtProc(proc, true)
	})
	// listen proc exist
	_, err = c.procsService.ConnectExitProc(func(execPath string, cgroup2Path string, pid string) {
		// get cgroup member
		cgroup := c.GetCGroupMember(execPath)
		if cgroup == nil {
			logger.Debugf("exe [%s] cant found in any cgroup", execPath)
			return
		}
		proc := ProcMessage{
			execPath:    execPath,
			pid:         pid,
			cgroup2Path: cgroup2Path,
		}
		// kernel delete proc
		err = cgroup.delCrtProc(proc, false)
	})
	return nil
}

func init() {
	logger = log.NewLogger("daemon/cgroup")
	logger.SetLogLevel(log.LevelDebug)
}
