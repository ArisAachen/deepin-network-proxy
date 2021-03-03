package NewCGroups

import (
	com "github.com/DeepinProxy/Com"
	netlink "github.com/linuxdeepin/go-dbus-factory/com.deepin.system.procs"
	"path/filepath"
	"reflect"
)

// cgroup2 main path
const (
	cgroup2Path = "/sys/fs/cgroup/unified"
	suffix      = ".slice"
	procsPath   = "cgroup.procs"
)

// source controller
type Controller struct {
	// controller name
	Name string // main app global

	// manager
	manager *Manager

	// control app exe path
	CtlPathSl []string

	// current control app message
	CtlProcMap map[string][]*netlink.ProcMessage
}

// add control app path
func (c *Controller) AddCtlAppPath(path string) {
	ifc, update, err := com.MegaAdd(c.CtlPathSl, path)
	if err != nil || !update {
		return
	}
	temp, ok := ifc.([]string)
	if !ok {
		return
	}
	c.CtlPathSl = temp
}

// del app path
func (c *Controller) DelCtlAppPath(path string) {
	ifc, update, err := com.MegaDel(c.CtlPathSl, path)
	if err != nil || !update {
		return
	}
	temp, ok := ifc.([]string)
	if !ok {
		return
	}
	c.CtlPathSl = temp
}

// check control app path exist
func (c *Controller) CheckCtlPathSl(path string) bool {
	for _, elem := range c.CtlPathSl {
		if elem == path {
			return true
		}
	}
	return false
}

// check if current control proc exist
func (c *Controller) CheckCtlProcExist(proc *netlink.ProcMessage) bool {
	// check map
	procSl, ok := c.CtlProcMap[proc.ExecPath]
	if !ok {
		return false
	}
	// check exist
	for _, elem := range procSl {
		if reflect.DeepEqual(elem, proc) {
			return true
		}
	}
	// not found
	return false
}

// add current control proc
func (c *Controller) AddCtrlProc(proc *netlink.ProcMessage) error {
	// check if exist
	if c.CheckCtlProcExist(proc) {
		return nil
	}
	// attach pid to cgroup
	err := attach(proc.Pid, c.GetControlPath())
	if err != nil {
		return err
	}
	// check if is nil
	if c.CtlProcMap[proc.ExecPath] == nil {
		c.CtlProcMap[proc.ExecPath] = []*netlink.ProcMessage{}
	}
	c.CtlProcMap[proc.ExecPath] = append(c.CtlProcMap[proc.ExecPath], proc)
	return nil
}

// delete current control proc
func (c *Controller) DelCtlProc(proc *netlink.ProcMessage, move bool) error {
	// check if exist
	if !c.CheckCtlProcExist(proc) {
		return nil
	}
	// not move to other cgroup, should attach to origin cgroup
	if !move {
		// attach pid to origin cgroup
		err := attach(proc.Pid, proc.CGroupPath)
		if err != nil {
			return err
		}
	}
	procSl := c.CtlProcMap[proc.ExecPath]
	// delete proc from self
	ifc, update, err := com.MegaDel(procSl, proc)
	if err != nil || update {
		return nil
	}
	temp, ok := ifc.([]*netlink.ProcMessage)
	if !ok {
		return nil
	}
	c.CtlProcMap[proc.ExecPath] = temp
	return nil
}

// /sys/fs/cgroup/unified/App.slice/cgroup.procs
func (c *Controller) GetControlPath() string {
	return filepath.Join(c.GetCGroupPath(), procsPath)
}

// /sys/fs/cgroup/unified/App.slice
func (c *Controller) GetCGroupPath() string {
	return filepath.Join(cgroup2Path, c.GetName())
}

// App.slice
func (c *Controller) GetName() string {
	return c.Name + suffix
}
