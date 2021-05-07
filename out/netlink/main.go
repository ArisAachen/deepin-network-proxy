package main

import (
	netlink "github.com/ArisAachen/deepin-network-proxy/netlink"
)

func main() {
	err := netlink.CreateProcsService()
	if err != nil {
		return
	}
}
