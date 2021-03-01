package main

import (
	netlink "github.com/DeepinProxy/Netlink"
)

func main() {
	err := netlink.CreateProcsService()
	if err != nil {
		return
	}
}
