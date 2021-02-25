package main

import (
	bus "github.com/DeepinProxy/DBus"
	netlink "github.com/DeepinProxy/Netlink"
	"time"
)

func main() {
	go func() {
		err := bus.CreateProxyService()
		if err != nil {
			return
		}
	}()

	go func() {
		err := netlink.CreateProcsService()
		if err != nil {
			return
		}
	}()

	time.Sleep(24 * time.Hour)
}
