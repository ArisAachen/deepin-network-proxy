package main

import (
	bus "github.com/DeepinProxy/DBus"
)

func main() {
	//uid := os.Getuid()
	//pid := os.Getpid()
	//time, err := com.GetProcStartTime(uint32(pid))
	//if err != nil {
	//	logger.Fatal(err)
	//}

	//// promote privilege
	//err = com.PromotePrivilege(com.ProxyActionId, uint32(uid), uint32(pid), time)
	//if err != nil {
	//	logger.Fatal(err)
	//}

	err := bus.CreateProxyService()
	if err != nil {
		return
	}
}
