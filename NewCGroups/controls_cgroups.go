package NewCGroups

import (
	"os"
	"path/filepath"
	"reflect"

	com "github.com/DeepinProxy/Com"
	define "github.com/DeepinProxy/Define"
	netlink "github.com/linuxdeepin/go-dbus-factory/com.deepin.system.procs"
)

// cgroup2 main path
const (
	cgroup2Path = "/sys/fs/cgroup/unified"
	suffix      = ".slice"
	procsPath   = "cgroup.procs"
)

type ControlProcSl []*netlink.ProcMessage

// get len
func (ctSl *ControlProcSl) Len() int {
	return len(*ctSl)
}

// attach pid back to origin cgroup
func (ctSl *ControlProcSl) Release() error {
	for _, ctrl := range *ctSl {
		err := attach(ctrl.Pid, ctrl.CGroupPath)
		if err != nil {
			logger.Warningf("[%s] attach %s back to origin cgroups %s failed, err: %v", ctrl.ExecPath, ctrl.Pid, ctrl.CGroupPath, err)
			return err
		}
		logger.Debugf("[%s] attach %s back to origin cgroups %s success", ctrl.ExecPath, ctrl.Pid, ctrl.CGroupPath)
	}
	return nil
}

// attach pid to new cgroups
func (ctSl *ControlProcSl) Attach(path string) error {
	for _, ctrl := range *ctSl {
		err := attach(ctrl.Pid, path)
		if err != nil {
			logger.Warningf("[%s] attach %s back to new cgroups %s failed, err: %v", ctrl.ExecPath, ctrl.Pid, path, err)
			return err
		}
		logger.Debugf("[%s] attach %s back to new cgroups %s success", ctrl.ExecPath, ctrl.Pid, path)
	}
	return nil
}

//// delete current control proc
//func (c *Controller) DelCtlProc(proc *netlink.ProcMessage, move bool) error {
//	// check if exist
//	if !c.CheckCtlProcExist(proc) {
//		return nil
//	}
//	// not move to other cgroup, should attach to origin cgroup
//	if !move {
//		// attach pid to origin cgroup
//		err := attach(proc.Pid, proc.CGroupPath)
//		if err != nil {
//			return err
//		}
//	}
//	procSl := c.CtlProcMap[proc.ExecPath]
//	// delete proc from self
//	ifc, update, err := com.MegaDel(procSl, proc)
//	if err != nil || update {
//		return nil
//	}
//	temp, ok := ifc.(ControlProcSl)
//	if !ok {
//		return nil
//	}
//	c.CtlProcMap[proc.ExecPath] = temp
//	return nil
//}

// del ctrl proc
//func (ctSl *ControlProcSl) DelCtrlProc(proc *netlink.ProcMessage) {
//	// check if exist
//	if !ctSl.CheckCtlProcExist(proc) {
//		return
//	}
//	// delete proc from self
//	_, update, err := com.MegaDel(*ctSl, proc)
//	if err != nil || update {
//		return
//	}
//}

// check if proc already exist
func (ctSl *ControlProcSl) CheckCtlProcExist(proc *netlink.ProcMessage) bool {
	for _, ctrl := range *ctSl {
		if reflect.DeepEqual(ctrl, proc) {
			return true
		}
	}
	return false
}

// source controller
type Controller struct {
	// controller name
	Name define.Scope // main app global

	// Fuzzy Priority
	Priority define.Priority

	// manager
	manager *Manager

	// control app exe path
	CtlPathSl []string

	// current control app message
	CtlProcMap map[string]ControlProcSl
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

// clear app ctl path
func (c *Controller) ClearCtlAppPath() {
	c.CtlPathSl = []string{}
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
		if reflect.DeepEqual(*elem, *proc) {
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

// move lower priority proc in
func (c *Controller) UpdateFromManagerAll() error {
	var lower bool
	for index := 0; index < c.manager.GetControllerCount(); index++ {
		// check if	is the same
		if lower {
			controller := c.manager.controllers[index]
			err := c.MoveToController(controller)
			if err != nil {
				logger.Warningf("[%s] update proc failed, err: %v", err)
				return err
			}
			logger.Warningf("[%s] update proc success, err: %v", err)
		}
		// found index lower, after here, is lower than now controller
		if c.manager.controllers[index] == c {
			lower = true
		}
	}
	return nil
}

// move lower priority proc in
func (c *Controller) UpdateFromManager(path string) error {
	controller := c.manager.GetControllerByCtlPath(path)
	// check if controller exist
	if controller != nil {
		// dont remove, because current priority is higher
		if controller.Priority >= c.Priority {
			logger.Debug("[%s] dont need update procs %s, %s has higher priority", c.Name, path, controller.Priority)
			return nil
		}
		procSl := controller.MoveOut(path)
		// check length
		if len(procSl) == 0 {
			return nil
		}
		err := c.MoveIn(path, procSl)
		if err != nil {
			return err
		}
		return nil
	}
	logger.Debugf("[%s] dont need update procs %s, cant find any controller", c.Name, path)
	return nil
}

// release all proc from controller, that may happen when stop controller
func (c *Controller) ReleaseAll() error {
	logger.Debugf("[%s] start release all procs", c.Name)
	// range all
	for _, ctrlPath := range c.CtlPathSl {
		err := c.ReleaseToManager(ctrlPath)
		if err != nil {
			return err
		}
	}
	// remove dir
	err := os.RemoveAll(c.GetCGroupPath())
	if err != nil {
		logger.Warning("[%s] remove cgroups path %s failed, err: %v", c.Name, c.GetCGroupPath(), err)
		return err
	}

	logger.Debugf("[%s] release all procs success", c.Name)
	return nil
}

// move now proc to lower controller or to default cgroups
func (c *Controller) ReleaseToManager(path string) error {
	logger.Debugf("[%s] start release %s", c.Name, path)
	// in case get self, clear self control path
	c.DelCtlAppPath(path)
	// get new procs
	procSl := c.MoveOut(path)
	// check if has elem
	if procSl.Len() == 0 {
		logger.Debugf("[%s] release has not control path procs %s", c.Name, path)
		return nil
	}
	// check if controller exist, now usually get lower priority path
	controller := c.manager.GetControllerByCtlPath(path)
	// path dont exist in any controller, attach back to origin cgroups
	if controller == nil {
		logger.Debugf("[%s] release has no lower priority, release to origin cgroup", c.Name)
		err := procSl.Release()
		if err != nil {
			logger.Warningf("[%s] release to origin cgroups failed, err: %v", c.Name, err)
			return err
		}
		return nil
	}
	// get controller is the highest one in the rest
	err := controller.MoveIn(path, procSl)
	if err != nil {
		return err
	}
	logger.Debugf("[%s] release %s success", c.Name, path)
	return nil
}

// move to control procs
func (c *Controller) MoveToController(controller *Controller) error {
	// compare priority
	if c.Priority >= controller.Priority {
		logger.Debugf("[%s] dont need to move %s, priority is higher", c.Name, controller.Name)
		return nil
	}
	// find control path
	for _, ctrlPath := range controller.CtlPathSl {
		// move proc out here
		procSl := c.MoveOut(ctrlPath)
		if procSl == nil {
			continue
		}
		err := controller.MoveIn(ctrlPath, procSl)
		if err != nil {
			return err
		}
	}
	return nil
}

// move in control procs
func (c *Controller) MoveIn(path string, inCtSl ControlProcSl) error {
	// check if exist control procs
	ognCtSl, ok := c.CtlProcMap[path]
	// if not, create one
	if !ok {
		// change old cgroups to new
		err := inCtSl.Attach(c.GetControlPath())
		if err != nil {
			return err
		}
		// save
		c.CtlProcMap[path] = inCtSl
		logger.Debugf("[%s] attach all to new cgroups", c.Name)
		return nil
	}
	// change and add
	for _, ctrl := range inCtSl {
		// check if already exist
		if com.MegaExist(ognCtSl, ctrl) {
			logger.Debugf("[%s] proc %v already exist in cgroups", c.Name, ctrl)
			continue
		}
		// if not exist, add in
		err := c.AddCtrlProc(ctrl)
		if err != nil {
			logger.Warningf("[%s] add %v to cgroups failed, err: %v", c.Name, err)
			return err
		}
	}
	logger.Debugf("[%s] attach all to new cgroups", c.Name)
	return nil
}

// move out control procs
func (c *Controller) MoveOut(path string) ControlProcSl {
	// check is exist control procs
	ctSl, ok := c.CtlProcMap[path]
	if !ok {
		logger.Debugf("[%s] has not control app path %s", c.Name, path)
		return nil
	}
	// delete from self
	delete(c.CtlProcMap, path)
	logger.Debugf("[%s] has not control app path %s", c.Name, path)
	return ctSl
}

// delete current control proc
func (c *Controller) DelCtlProc(proc *netlink.ProcMessage) error {
	// check if exist
	if !c.CheckCtlProcExist(proc) {
		return nil
	}
	procSl := c.CtlProcMap[proc.ExecPath]
	// delete proc from self
	ifc, update, err := com.MegaDel(procSl, proc)
	if err != nil || update {
		return nil
	}
	temp, ok := ifc.(ControlProcSl)
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
	return c.Name.ToString() + suffix
}
