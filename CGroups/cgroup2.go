package CGroups

import (
	"errors"
	"fmt"
	"github.com/godbus/dbus"
	"path/filepath"
	"pkg.deepin.io/lib/log"
	"reflect"
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
	procs = "cgroup.procs"

	// suffix
	suffix = ".slice"

	// maintain path
	MainGRP = "main"

	// sub path
	appProxyPath      = "app-proxy.slice"
	globalNoProxyPath = "global-no-proxy.slice"
)

// cgroup priority
const (
	MainLevel = iota + 1
	AppProxyLevel
	GlobalProxyLevel
)

// proc message
type ProcMessage struct {
	execPath    string // exe path
	cgroup2Path string // mark origin cgroup v2 path
	pid         string // pid
}

// cgroup
type CGroupMember struct {
	parent *CGroupManager

	priority int

	// name
	path string // main-proc.slice ...

	// target exe path slice,   /usr/sbin/NetworkManager
	tgtExeSl []string //

	// current process map,  [/usr/sbin/NetworkManager][1 12256]
	procMap map[string][]ProcMessage
}

func newGrpMsg() *CGroupMember {
	return &CGroupMember{
		tgtExeSl: []string{},
		procMap:  make(map[string][]ProcMessage),
	}
}

// get cgroup tgtExeSl path  as /sys/fs/cgroup/unified/main-proc.slice/cgroup.procs
func (p *CGroupMember) getProcsPath() string {
	path := filepath.Join(p.getProcsDir(), procs)
	return path
}

func (p *CGroupMember) getProcsDir() string {
	return filepath.Join(cgroup2Path, p.path+suffix)
}

// add module exec paths
func (p *CGroupMember) addTgtExes(exePaths []string) {
	for _, path := range exePaths {
		p.addTgtExe(path)
	}
}

// add module exec path
func (p *CGroupMember) addTgtExe(exePath string) {
	// check if already exist
	ifc, update, err := com.MegaAdd(p.tgtExeSl, exePath)
	if err != nil {
		logger.Warningf("[%s] add tgtExe [%s] failed, err: %v", p.path, exePath, err)
		return
	}
	// check if need update
	if !update {
		logger.Debugf("[%s] dont add tgtExe [%s], already exist", p.path, exePath)
		return
	}
	// check if type correct
	strSl, ok := ifc.([]string)
	if !ok {
		logger.Warningf("[%s] add tgtExe [%s] failed, ifc is not string slice", p.path, exePath)
		return
	}
	// recover
	p.tgtExeSl = strSl
	logger.Debugf("[%s] add tgtExe [%s] failed success", p.path, exePath)
}

// del module tgtExeSl paths
func (p *CGroupMember) delTgtExes(exePaths []string) {
	for _, path := range exePaths {
		p.delTgtExe(path)
	}
}

// del module exec path
func (p *CGroupMember) delTgtExe(exePath string) {
	// check if already exist
	ifc, update, err := com.MegaDel(p.tgtExeSl, exePath)
	if err != nil {
		logger.Warningf("[%s] delete tgtExe [%s] failed, err: %v", p.path, exePath, err)
		return
	}
	// check if need update
	if !update {
		logger.Debugf("[%s] dont need delete tgtExe [%s], not exist", p.path, exePath)
		return
	}
	// check if type correct
	strSl, ok := ifc.([]string)
	if !ok {
		logger.Warningf("[%s] delete tgtExe [%s] failed, ifc is not string slice", p.path, exePath)
		return
	}
	// recover
	p.tgtExeSl = strSl
	logger.Debugf("[%s] delete tgtExe [%s] success", p.path, exePath)
}

// check if exe path exist
func (p *CGroupMember) existTgtExe(exePath string) bool {
	for _, exe := range p.tgtExeSl {
		if exe == exePath {
			logger.Debugf("[%s] tgtExe [%s] found", p.path, exePath)
			return true
		}
	}
	logger.Debugf("[%s] tgtExe [%s] not found", exePath, p.path)
	return false
}

// add proc to cgroup
func (p *CGroupMember) addCrtProc(proc ProcMessage, active bool) error {
	// check if key exist, if not , add key
	procSl, ok := p.procMap[proc.execPath]
	if !ok {
		logger.Debugf("[%s] add crtProc dont include exe [%s]", p.path, proc.execPath)
		procSl = []ProcMessage{}
	}
	// try to add
	ifc, update, err := com.MegaAdd(procSl, proc)
	if err != nil {
		logger.Warningf("[%s] add crtProc [%v] failed, err: %v", p.path, proc, err)
		return err
	}
	// if update
	if !update {
		logger.Debugf("[%s] dont need add crtProc [%v], already exist", p.path, proc)
		return nil
	}
	// check if type correct
	procSl, ok = ifc.([]ProcMessage)
	if !ok {
		logger.Warningf("[%s] add crtProc [%v] failed, ifc is ProcMessage slice", p.path, proc)
		return errors.New("ifc is not match")
	}
	// add pid to sl
	p.procMap[proc.execPath] = procSl

	if active {
		err = AddCGroup(proc.pid, p.getProcsPath())
	}
	return nil
}

// del proc from cgroup
func (p *CGroupMember) delCrtProc(proc ProcMessage, active bool) error {
	// check if key exist, if not , add key
	procSl, ok := p.procMap[proc.execPath]
	if !ok {
		logger.Debugf("[%s] delete crtProc dont include exe [%s]", p.path, proc.execPath)
		return nil
	}
	// try to add
	ifc, update, err := com.MegaDel(procSl, proc)
	if err != nil {
		logger.Warningf("[%s] delete crtProc [%v] failed, err: %v", p.path, proc, err)
		return err
	}
	// if update
	if !update {
		logger.Debugf("[%s] dont need delete crtProc [%v], not exist", p.path, proc)
		return nil
	}
	// check if type correct
	procSl, ok = ifc.([]ProcMessage)
	if !ok {
		logger.Warningf("[%s] delete crtProc [%v] failed, ifc is ProcMessage slice", p.path, proc)
		return errors.New("ifc is not match")
	}
	// add pid to sl
	p.procMap[proc.execPath] = procSl
	// del from cgroup
	if active {

	}
	return nil
}

// find pid and return index
func (p *CGroupMember) existProc(proc ProcMessage) bool {
	// check if length is 0
	if len(p.procMap) == 0 {
		logger.Debugf("[%s] check proc [%v] not exist, map length is 0", p.path, proc)
		return false
	}
	// proc slice
	procSl, ok := p.procMap[proc.execPath]
	if !ok {
		logger.Debugf("[%s] check proc [%v] not exist, slice length is 0", p.path, proc)
		return false
	}
	// check range
	for _, elem := range procSl {
		if reflect.DeepEqual(elem, proc) {
			logger.Debugf("[%s] check proc [%v] exist", p.path, proc)
			return true
		}
	}
	logger.Debugf("[%s] check proc [%v] not exist", p.path, proc)
	return false
}

// cgroup v2 /sys/fs/cgroup/unified
type CGroupManager struct {
	CGroups []*CGroupMember // map[priority]CGroupMember

	// lock
	lock sync.Mutex
}

// create CGroupManager
func NewCGroupManager() *CGroupManager {
	return &CGroupManager{
		CGroups: []*CGroupMember{},
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

func (c *CGroupManager) GetCGroupMessage(exe string) *CGroupMember {
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
	// listen proc exec
	_, err = procs.ConnectExecProc(func(execPath string, cgroup2Path string, pid string) {
		// get cgroup member
		cgroup := c.GetCGroupMessage(execPath)
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
	_, err = procs.ConnectExitProc(func(execPath string, cwdPath string, pid string) {
		// get cgroup member
		cgroup := c.GetCGroupMessage(execPath)
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
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
