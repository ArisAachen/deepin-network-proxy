package main

import (
	proxyDBus "github.com/DeepinProxy/DBus"
	"pkg.deepin.io/lib/log"
)

func main() {

	logger := log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelInfo)
	//
	manager := proxyDBus.NewManager()
	err := manager.Init()
	if err != nil {
		logger.Warningf("manager init failed, err: %v", err)
		return
	}
	// load config
	_ = manager.LoadConfig()
	//if err != nil {
	//	log.Fatal(err)
	//}
	// export dbus service
	err = manager.Export()
	if err != nil {
		logger.Warningf("manager export failed, err: %v", err)
		return
	}
	// wait
	manager.Wait()
}
