package DBus

import (
	newCGroups "github.com/DeepinProxy/NewCGroups"
)

//// create cgroup handler add to manager
//func (mgr *proxyPrv) CreateCGroupController(manager *newCGroups.Manager) *newCGroups.Controller {
//	// controller := manager.CreatePriorityController(mgr.scope.String(),mgr.getDBusPath())
//}


func (mgr *proxyPrv) setController(controller *newCGroups.Controller) {
	mgr.controller = controller
}
