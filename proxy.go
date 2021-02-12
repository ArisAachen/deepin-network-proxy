package main

import (
	com "github.com/DeepinProxy/Com"
	tProxy "github.com/DeepinProxy/TProxy"
	"os"
	"pkg.deepin.io/lib/dbusutil"
	"pkg.deepin.io/lib/log"
)

var logger = log.NewLogger("daemon/proxy")
var handlerMgr *tProxy.HandlerMgr

func main() {
	uid := os.Getuid()
	pid := os.Getpid()
	time, err := com.GetProcStartTime(uint32(pid))
	if err != nil {
		logger.Fatal(err)
	}

	// promote privilege
	err = com.PromotePrivilege(com.ProxyActionId, uint32(uid), uint32(pid), time)
	if err != nil {
		logger.Fatal(err)
	}

	// system service
	service, err := dbusutil.NewSessionService()
	if err != nil {
		logger.Fatalf("create system service failed, err: %v", err)
	}
	// create proxy manager object
	proxyManager := NewProxyManager()
	err = service.Export(DBusPath, proxyManager)
	if err != nil {
		logger.Fatalf("export DBus path failed, err: %v", err)
	}
	err = service.RequestName(DBusServiceName)
	if err != nil {
		logger.Fatalf("request DBus name failed, err: %v", err)
	}
	service.Wait()
}
