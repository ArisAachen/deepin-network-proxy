package CGroups

import (
	"fmt"
	"path/filepath"
	"pkg.deepin.io/lib/log"
	"sort"
	"sync"

	com "github.com/DeepinProxy/Com"
)

var logger *log.Logger

// cgroup v2 cgroup.procs

const (
	// cgroup2 main path
	cgroup2Path = "/sys/fs/cgroup/unified"

	// cgroup procs
	procs = "cgroup.procs"

	// suffix
	suffix = ".slice"

	// maintain path
	MainGRP = "main"

	// sub path
	appProxyPath      = "app-proxy.slice"
	globalNoProxyPath = "global-no-proxy.slice"
)

// cgroup
type cgroupMessage struct {
	priority int

	// name
	path string // main-proc.slice ...

	// proc path slice
	procs []string // /usr/sbin/NetworkManager
}

func newGrpMsg() *cgroupMessage {
	return &cgroupMessage{
		procs: []string{},
	}
}

// get cgroup procs path  as /sys/fs/cgroup/unified/main-proc.slice/cgroup.procs
func (p *cgroupMessage) getProcsPath() string {
	path := filepath.Join(p.getProcsDir(), procs)
	return path
}

func (p *cgroupMessage) getProcsDir() string {
	return filepath.Join(cgroup2Path, p.path+suffix)
}

// add module exec paths
func (p *cgroupMessage) addProcs(exePaths []string) {
	for _, path := range exePaths {
		p.addProc(path)
	}
}

// add module exec path
func (p *cgroupMessage) addProc(exePath string) {
	// check if len is nil
	if len(p.procs) == 0 {
		p.procs = append(p.procs, exePath)
		return
	}
	// check if already exist
	index := sort.SearchStrings(p.procs, exePath)
	if index == len(p.procs) {
		// p.procs[index] = exePath
		p.procs = append(p.procs, exePath)
		logger.Debugf("Proc [%s] added success in cgroupMessage [%s]", exePath, p.path)
		return
	}
	logger.Debugf("Proc [%s] added failed, already exist in cgroupMessage [%s]", exePath, p.path)
}

// del module exe paths
func (p *cgroupMessage) delProcs(exePaths []string) {
	for _, path := range exePaths {
		p.delProc(path)
	}
}

// del module exec path
func (p *cgroupMessage) delProc(exePath string) {
	// check if already exist
	index := sort.SearchStrings(p.procs, exePath)
	if index == len(p.procs) {
		logger.Debugf("Proc [%s] delete failed, not exist in cgroupMessage [%s]", exePath, p.path)
		return
	}
	logger.Debugf("Proc [%s] delete success from cgroupMessage [%s]", exePath, p.path)
}

func (p *cgroupMessage) existProc(exePath string) bool {
	index := sort.SearchStrings(p.procs, exePath)
	if index < len(p.procs) {
		logger.Debugf("Proc [%s] found in cgroupMessage [%s]", exePath, p.path)
		return true
	}
	logger.Debugf("Proc [%s] not found in cgroupMessage [%s]", exePath, p.path)
	return false
}

// cgroup v2 /sys/fs/cgroup/unified
type CGroupManager struct {
	CGroups []cgroupMessage // map[priority]cgroupMessage

	// lock
	lock sync.Mutex
}

// create CGroupManager
func NewCGroupManager() *CGroupManager {
	return &CGroupManager{
		CGroups: []cgroupMessage{},
	}
}

// create cgroup path
func (c *CGroupManager) CreateCGroup(level int, elemPath string) error {
	// lock
	c.lock.Lock()
	defer c.lock.Unlock()
	// check if cgroup already exist
	for _, cgroup := range c.CGroups {
		if cgroup.path == elemPath {
			logger.Warningf("create group failed, path [%s] already exist", elemPath)
			return fmt.Errorf("create group failed, path [%s] already exist", elemPath)
		}
		if cgroup.priority == level {
			logger.Warningf("create group failed, level [%d] already exist", level)
			return fmt.Errorf("create group failed, level [%d] already exist", level)
		}
	}
	// use level to mark priority
	cgpMsg := cgroupMessage{
		path:     elemPath,
		priority: level,
		procs:    []string{},
	}
	// add to manager
	c.CGroups = append(c.CGroups, cgpMsg)
	// sort slice
	sort.SliceStable(c.CGroups, func(i, j int) bool {
		// check if priority is sorted correctly
		if c.CGroups[i].priority > c.CGroups[j].priority {
			return false
		}
		return true
	})

	// make dir
	cgpProcs := cgpMsg.getProcsPath()
	err := com.GuaranteeDir(cgpProcs)
	if err != nil {
		logger.Warningf("mkdir [%s] failed, err: %v", cgpProcs, err)
		return err
	}
	return nil
}

// add proc path to cgroup elem
func (c *CGroupManager) AddCGroupProcs(elem string, procs []string) {
	// lock
	c.lock.Lock()
	defer c.lock.Unlock()

	// add
	for _, cgroup := range c.CGroups {
		if cgroup.path == elem {
			logger.Debugf("cgroup [%s] found in manager, begin to add procs [%s]", elem, procs)
			cgroup.addProcs(procs)
			return
		}
	}
	// if not found, out put log
	logger.Warningf("cgroup [%s] not found in manager", elem)
}

// add proc path to cgroup elem
func (c *CGroupManager) DelCGroupProcs(elem string, procs []string) {
	// lock
	c.lock.Lock()
	defer c.lock.Unlock()

	// add
	for _, cgroup := range c.CGroups {
		if cgroup.path == elem {
			logger.Debugf("cgroup [%s] found in manager, begin to del proc [%s]", elem, procs)
			cgroup.delProcs(procs)
			return
		}
	}
	// if not found, out put log
	logger.Warningf("cgroup [%s] not found in manager", elem)
}

func (c *CGroupManager) GetCGroupProcsPath(proc string) string {
	// lock
	c.lock.Lock()
	defer c.lock.Unlock()

	// search which cgroup, proc exist
	for _, cgroup := range c.CGroups {
		if cgroup.existProc(proc) {
			return cgroup.getProcsPath()
		}
	}
	logger.Warningf("proc [%s] cant found in any cgroup", proc)
	// if not found
	return ""
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
