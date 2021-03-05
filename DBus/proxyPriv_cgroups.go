package DBus

import (
	com "github.com/DeepinProxy/Com"
	define "github.com/DeepinProxy/Define"
)

func (mgr *proxyPrv) getCGroupPriority() define.Priority {
	return mgr.priority
}

// create cgroup handler add to manager
func (mgr *proxyPrv) createCGroupController() error {
	controller, err := mgr.manager.controllerMgr.CreatePriorityController(mgr.scope, mgr.priority)
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
	// add map
	for path, procSl := range procsMap {
		// check if exist
		if !com.MegaExist(mgr.Proxies.ProxyProgram, path) {
			logger.Debugf("[%s] dont need add %s at first", mgr.scope, path)
			continue
		}

		// check if already exist
		controller := mgr.manager.controllerMgr.GetControllerByCtlPath(path)
		if controller == nil {
			// add path
			mgr.controller.AddCtlAppPath(path)
			err := mgr.controller.MoveIn(path, procSl)
			if err != nil {
				logger.Warning("[%s] add procs %s at first failed, err: %v", mgr.scope, path, err)
				continue
			}
			logger.Debugf("[%s] add procs %s at first success", mgr.scope, path)
		} else {
			err = mgr.controller.UpdateFromManager(path)
			if err != nil {
				logger.Warning("[%s] add proc %s from %s at first failed, err: %v", mgr.scope, path, controller.Name, err)
			} else {
				logger.Debugf("[%s] add proc %s from %s at first failed", mgr.scope, path, controller.Name)
			}
			mgr.controller.AddCtlAppPath(path)
		}
	}

	return nil
}
