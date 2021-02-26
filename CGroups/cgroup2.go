package CGroups

import (
	"fmt"
	"github.com/godbus/dbus"
	"path/filepath"
	"pkg.deepin.io/lib/log"
	"sort"
	"sync"

	com "github.com/DeepinProxy/Com"
	netlink "github.com/linuxdeepin/go-dbus-factory/com.deepin.system.procs"
)

var logger *log.Logger

// cgroup v2 cgroup.tgtExeSl

const (
	// cgroup2 main path
	cgroup2Path = "/sys/fs/cgroup/unified"

	// cgroup tgtExeSl
	procs = "cgroup.tgtExeSl"

	// suffix
	suffix = ".slice"

	// maintain path
	MainGRP = "main"

	// sub path
	appProxyPath      = "app-proxy.slice"
	globalNoProxyPath = "global-no-proxy.slice"
)

// proc message
type ProcMessage struct {
	execPath    string // exe path
	cgroup2Path string // mark origin cgroup v2 path
	pid         string // pid
}

// cgroup
type cgroupMessage struct {
	priority int

	// name
	path string // main-proc.slice ...

	// target exe path slice
	tgtExeSl []string // /usr/sbin/NetworkManager

	// current pid slice 1 12256
	pidSl  []string
	procSl []ProcMessage
}

func newGrpMsg() *cgroupMessage {
	return &cgroupMessage{
		tgtExeSl: []string{},
	}
}

// get cgroup tgtExeSl path  as /sys/fs/cgroup/unified/main-proc.slice/cgroup.tgtExeSl
func (p *cgroupMessage) getProcsPath() string {
	path := filepath.Join(p.getProcsDir(), procs)
	return path
}

func (p *cgroupMessage) getProcsDir() string {
	return filepath.Join(cgroup2Path, p.path+suffix)
}

// add module exec paths
func (p *cgroupMessage) addTgtExes(exePaths []string) {
	for _, path := range exePaths {
		p.addTgtExe(path)
	}
}

// add module exec path
func (p *cgroupMessage) addTgtExe(exePath string) {
	// check if len is nil
	if len(p.tgtExeSl) == 0 {
		p.tgtExeSl = append(p.tgtExeSl, exePath)
		return
	}
	// check if already exist
	index := sort.SearchStrings(p.tgtExeSl, exePath)
	if index == len(p.tgtExeSl) {
		// p.tgtExeSl[index] = exePath
		p.tgtExeSl = append(p.tgtExeSl, exePath)
		logger.Debugf("Proc [%s] added success in cgroupMessage [%s]", exePath, p.path)
		return
	}
	logger.Debugf("Proc [%s] added failed, already exist in cgroupMessage [%s]", exePath, p.path)
}

// del module tgtExeSl paths
func (p *cgroupMessage) delTgtExes(exePaths []string) {
	for _, path := range exePaths {
		p.delTgtExe(path)
	}
}

// del module exec path
func (p *cgroupMessage) delTgtExe(exePath string) {
	// check if already exist
	index := sort.SearchStrings(p.tgtExeSl, exePath)
	if index == len(p.tgtExeSl) {
		logger.Debugf("Proc [%s] delete failed, not exist in cgroupMessage [%s]", exePath, p.path)
		return
	}
	logger.Debugf("Proc [%s] delete success from cgroupMessage [%s]", exePath, p.path)
}

// check if exe path exist
func (p *cgroupMessage) existTgtExe(exePath string) bool {
	for _, exe := range p.tgtExeSl {
		if exe == exePath {
			logger.Debugf("Proc [%s] found in cgroupMessage [%s]", exePath, p.path)
			return true
		}
	}
	logger.Debugf("Proc [%s] not found in cgroupMessage [%s]", exePath, p.path)
	return false
}

// add pid to cgroup
func (p *cgroupMessage) addCrtProc(proc ProcMessage) error {
	// check if pid is already exist
	exist, _ := p.existPid(proc.pid)
	if exist {
		logger.Debugf("dont need to add pid, proc [%v] already exist in [%s]", proc, p.path)
		return nil
	}
	// add pid to sl
	p.procSl = append(p.procSl, proc)
	return nil
}

// del pid from cgroup
func (p *cgroupMessage) delCrtProc(proc ProcMessage) error {
	// check if pid is already exist
	exist, index := p.existPid(proc.pid)
	if !exist {
		logger.Debugf("dont need to del proc, proc [%s] dont exist in [%s]", proc, p.path)
		return nil
	}
	// del pid from sl
	p.procSl = append(p.procSl[:index], p.procSl[index+1:]...)
	return nil
}

// find pid and return index
func (p *cgroupMessage) existPid(pid string) (bool, int) {
	// check if length is 0
	if len(p.procSl) == 0 {
		logger.Debugf("check exist, [%s] length is 0", p.path)
		return false, 0
	}
	// search pid
	index := sort.Search(len(p.procSl), func(i int) bool {
		if p.procSl[i].pid == pid {
			return false
		}
		return true
	})
	if index == len(p.procSl) {
		logger.Debugf("pid [%s] dont exist in [%s]", pid, p.path)
		return false, 0
	}
	// found
	logger.Debugf("pid [%s] dont exist in [%s]", pid, p.path)
	return false, index
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
		tgtExeSl: []string{},
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
			logger.Debugf("cgroup [%s] found in manager, begin to add tgtExeSl [%s]", elem, procs)
			cgroup.addTgtExes(procs)
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
			cgroup.delTgtExes(procs)
			return
		}
	}
	// if not found, out put log
	logger.Warningf("cgroup [%s] not found in manager", elem)
}

func (c *CGroupManager) GetCGroupMessage(exe string) *cgroupMessage {
	// lock
	c.lock.Lock()
	defer c.lock.Unlock()

	// search which cgroup, proc exist
	for _, cgroup := range c.CGroups {
		if cgroup.existTgtExe(exe) {
			logger.Debugf("exe [%s] is found in [%s]", exe, cgroup.path)
			return &cgroup
		}
	}
	logger.Debugf("exe [%s] cant found in any cgroup", exe)
	return nil
}

func (c *CGroupManager) GetCGroupProcsPath(exe string) string {
	// search which cgroup, proc exist
	cgroup := c.GetCGroupMessage(exe)
	if cgroup == nil {
		return ""
	}
	// if found
	return cgroup.path
}

func (c *CGroupManager) Listen() error {
	systemBus, err := dbus.SystemBus()
	if err != nil {
		logger.Warningf("create system bus failed, err: %v", err)
		return err
	}
	procs := netlink.NewProcs(systemBus)
	_, err = procs.ConnectExecProc(func(execPath string, cwdPath string, pid string) {
		cgroup := c.GetCGroupMessage(execPath)
		if cgroup == nil {
			logger.Debugf("exe [%s] cant found in any cgroup", execPath)
			return
		}

	})

	return nil
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
