package UdpProxy

import "net"

type BaseHandler interface {
	Tunnel(rConn net.Conn, addr *net.UDPAddr) error
}


