package DBus

import define "github.com/DeepinProxy/Define"

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

func (mgr *proxyPrv) setManager(manager *Manager) {
	mgr.manager = manager
}
