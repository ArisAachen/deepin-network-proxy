package main

import (
	tProxy "github.com/DeepinProxy/TProxy"
	"pkg.deepin.io/lib/dbusutil"
	"pkg.deepin.io/lib/log"
)

var logger = log.NewLogger("daemon/proxy")
var handlerMgr *tProxy.HandlerMgr

func main() {

	// system service
	service, err := dbusutil.NewSystemService()
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
