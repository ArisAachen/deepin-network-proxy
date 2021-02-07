package main

import (
	cfg "github.com/DeepinProxy/Config"
	tProxy "github.com/DeepinProxy/TProxy"
	"pkg.deepin.io/lib/log"
)

var logger = log.NewLogger("daemon/proxy")
var mgr *tProxy.HandlerMgr

// proxy main
func main() {

	//var udp bool = true
	//var lsp string = ":8080"
	var proto string = "sock5"
	logger.SetLogLevel(log.LevelDebug)

	mgr = tProxy.NewHandlerMsg()

	// get config
	config := cfg.NewProxyCfg()
	err := config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
	if err != nil {
		logger.Warningf("load config failed, err: %v", err)
		return
	}

	proxy, err := config.GetProxy("global", proto)
	if err != nil {
		logger.Warningf("get proxy from config failed, err: %v", err)
		return
	}

	NewTcpProxy(":8080", tProxy.SOCK5TCP, proxy)
}
