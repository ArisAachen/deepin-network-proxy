package DBus

import (
	define "github.com/ArisAachen/deepin-network-proxy/define"
)

func (mgr *proxyPrv) getCGroupPriority() define.Priority {
	return mgr.priority
}

// create cgroup handler add to manager
func (mgr *proxyPrv) createCGroupController() error {
	controller, err := mgr.manager.controllerMgr.CreatePriorityController(mgr.scope, mgr.uid, mgr.priority)
	if err != nil {
		return err
	}
	mgr.controller = controller
	return nil
}

// first adjust cgroups
func (mgr *proxyPrv) firstAdjustCGroups() error {
	// get all procs message
	procsMap, err := mgr.manager.GetAllProcs()
	if err != nil {
		return err
	}

	// range map
	for _, path := range mgr.Proxies.ProxyProgram {
		// check if already exist
		controller := mgr.manager.controllerMgr.GetControllerByCtlPath(path)
		// controller already exist
		if controller != nil {
			err = mgr.controller.UpdateFromManager(path)
			if err != nil {
				logger.Warning("[%s] add proc %s from %s at first failed, err: %v", mgr.scope, path, controller.Name, err)
			} else {
				logger.Debugf("[%s] add proc %s from %s at first failed", mgr.scope, path, controller.Name)
			}

		} else {
			// not exist
			procSl, ok := procsMap[path]
			// if has current proc slice
			if ok {
				err := mgr.controller.MoveIn(path, procSl)
				if err != nil {
					logger.Warning("[%s] add procs %s at first failed, err: %v", mgr.scope, path, err)
					continue
				}
				logger.Debugf("[%s] add procs %s at first success", mgr.scope, path)
			}
		}
		// add path to path slice
		mgr.controller.AddCtlAppPath(path)
	}

	return nil
}
