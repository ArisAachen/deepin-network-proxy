package Com

import (
	"golang.org/x/sys/unix"
	"net"
	"syscall"
)

const (
	SoOriginalDst    = 80
	Ip6SoOriginalDst = 80 // from linux/include/uapi/linux/netfilter_ipv6/ip6_tables.h
)

// get origin destination addr
func GetTcpRemoteAddr(conn *net.TCPConn) (*net.TCPAddr, error) {
	// get file descriptor
	file, err := conn.File()
	if err != nil {
		return nil, err
	}
	fd := int(file.Fd())

	// from linux/include/uapi/linux/netfilter_ipv4.h
	req, err := unix.GetsockoptIPv6Mreq(fd, syscall.IPPROTO_IP, SoOriginalDst)
	if err != nil {
		return nil, err
	}

	// struct tcp addr
	tcpAddr := &net.TCPAddr{
		IP:   req.Multiaddr[4:8],
		Port: int(req.Multiaddr[2])<<8 + int(req.Multiaddr[3]),
	}
	return tcpAddr, nil
}
