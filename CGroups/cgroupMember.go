package CGroups

import (
	"errors"
	"path/filepath"
	"reflect"
	"sync"

	com "github.com/DeepinProxy/Com"
	netlink "github.com/linuxdeepin/go-dbus-factory/com.deepin.system.procs"
)

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

//// proc message
//type ProcMessage struct {
//	ExecPath   string // exe path
//	CGroupPath string // mark origin cgroup v2 path
//	Pid        string // pid
//}

// cgroup
type CGroupMember struct {
	// cgroup manager
	parent *CGroupManager

	// read write lock
	lock sync.RWMutex

	// priority level
	priority int

	// name
	path string // main-proc.slice ...

	// target exe path slice,   /usr/sbin/NetworkManager
	tgtExeSl []string //

	// current process map,  [/usr/sbin/NetworkManager][1 12256]
	procMap map[string][]netlink.ProcMessage
}

func newGrpMsg() *CGroupMember {
	return &CGroupMember{
		tgtExeSl: []string{},
		procMap:  make(map[string][]netlink.ProcMessage),
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

func (c *CGroupMember) GetCGroupPath() string {
	return c.path + suffix
}

// add module exec paths
func (p *CGroupMember) AddTgtExes(exePaths []string) {
	// lock
	p.lock.Lock()
	defer p.lock.Unlock()

	// add exr path
	for _, path := range exePaths {
		p.addTgtExe(path)
	}
}

// add module exec path
func (p *CGroupMember) addTgtExe(exePath string) {
	// check if allow add, according to level
	member := p.parent.GetCGroupMember(exePath)
	// already added
	if member != nil {
		// check priority
		if member.priority == p.priority {
			logger.Debugf("[%s] dont need add tgtExe [%s], in the same priority", p.path, exePath)
			return
		}
		// dont need to add, exe is hold by higher priority
		if member.priority > p.priority {
			logger.Debugf("[%s] dont need add tgtExe [%s], exist in higher priority", p.path, exePath)
			return
		}
		// lower priority need delete
		if member.priority < p.priority {
			logger.Debugf("[%s] delete tgtExe [%s] from lower priority [%s]", p.path, exePath, member.path)
			// del current procs, dont need to delete from cgroup
			// these will hang in new procs
			member.DelTgtExes([]string{exePath}, false)
		}
	}
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
	procMap, err := p.parent.getProcs()
	if err != nil {
		return
	}
	procSl, ok := procMap[exePath]
	if !ok {
		logger.Debugf("[%s] add tgtExe [%s] but has no exec", p.path, exePath)
	} else {
		for _, proc := range procSl {
			_ = p.addCrtProc(proc, true)
		}
	}
	logger.Debugf("[%s] add tgtExe [%s] success", p.path, exePath)
}

// del module tgtExeSl paths
func (p *CGroupMember) DelTgtExes(exePaths []string, active bool) {
	// unlock
	p.lock.Lock()
	defer p.lock.Unlock()

	// delete exe path
	for _, path := range exePaths {
		p.delTgtExe(path, active)
	}
}

// del module exec path
func (p *CGroupMember) delTgtExe(exePath string, active bool) {
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
	// delete current proc
	_ = p.delCrtProcs(exePath, active)
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
func (p *CGroupMember) addCrtProc(proc netlink.ProcMessage, active bool) error {
	// check if key exist, if not , add key
	procSl, ok := p.procMap[proc.ExecPath]
	if !ok {
		logger.Debugf("[%s] add crtProc dont include exe [%s]", p.path, proc.ExecPath)
		procSl = []netlink.ProcMessage{}
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
	procSl, ok = ifc.([]netlink.ProcMessage)
	if !ok {
		logger.Warningf("[%s] add crtProc [%v] failed, ifc is ProcMessage slice", p.path, proc)
		return errors.New("ifc is not match")
	}
	// add pid to sl
	p.procMap[proc.ExecPath] = procSl
	if active {
		err = AttachCGroup(p.getProcsPath(), proc.Pid)
	}
	return nil
}

// use to delete current procs
func (p *CGroupMember) delCrtProcs(exe string, active bool) error {
	procSl, ok := p.procMap[exe]
	if !ok {
		logger.Debugf("[%s] dont need to del CrtProcs, exe not exist", p.path)
		return nil
	}
	for _, proc := range procSl {
		_ = p.delCrtProc(proc, active)
	}
	return nil
}

// del proc from cgroup
func (p *CGroupMember) delCrtProc(proc netlink.ProcMessage, active bool) error {
	// check if key exist, if not , add key
	procSl, ok := p.procMap[proc.ExecPath]
	if !ok {
		logger.Debugf("[%s] delete crtProc dont include exe [%s]", p.path, proc.ExecPath)
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
	procSl, ok = ifc.([]netlink.ProcMessage)
	if !ok {
		logger.Warningf("[%s] delete crtProc [%v] failed, ifc is ProcMessage slice", p.path, proc)
		return errors.New("ifc is not match")
	}
	// add pid to sl
	p.procMap[proc.ExecPath] = procSl
	// attach pid to origin cgroup
	if active {
		err = AttachCGroup(proc.CGroupPath, proc.Pid)
	}
	return nil
}

// find pid and return index
func (p *CGroupMember) existProc(proc netlink.ProcMessage) bool {
	// check if length is 0
	if len(p.procMap) == 0 {
		logger.Debugf("[%s] check proc [%v] not exist, map length is 0", p.path, proc)
		return false
	}
	// proc slice
	procSl, ok := p.procMap[proc.ExecPath]
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
