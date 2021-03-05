package main

import (
	proxyDBus "github.com/DeepinProxy/DBus"
	"log"
)

func main() {

	//
	manager := proxyDBus.NewManager()
	err := manager.Init()
	if err != nil {
		log.Fatal(err)
	}
	// load config
	err = manager.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	// export dbus service
	err = manager.Export()
	if err != nil {
		log.Fatal(err)
	}
	// wait
	manager.Wait()
}
