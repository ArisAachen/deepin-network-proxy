package DBus

import (
	"net"
)

type fakeIP struct {
	start uint32
	end   uint32
	index uint32
}

func ipToUint(ip net.IP) uint32 {
	return (uint32(ip[0]) << 24) | (uint32(ip[1]) << 16) | (uint32(ip[2]) << 8) | uint32(ip[3])
}

func uintToIP(v uint32) net.IP {
	return net.IP{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

func prefixToMask(prefix uint32) uint32 {
	return uint32(0xFFFFFFFF << (32 - prefix))
}

func newFakeIP(ip net.IP, prefix uint32) fakeIP {
	ipUint := ipToUint(ip)
	mask := prefixToMask(prefix)

	start := ipUint & mask
	end := ipUint | ^mask

	return fakeIP{
		start: start,
		end:   end,
	}
}

func (i *fakeIP) new() net.IP {
	current := i.start + i.index
	if current > i.end {
		// todo:
	}

	i.index++
	return uintToIP(current)
}
