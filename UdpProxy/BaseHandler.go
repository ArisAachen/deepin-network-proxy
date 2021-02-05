package UdpProxy

import (
	"net"

	"github.com/DeepinProxy/Com"
	"pkg.deepin.io/lib/log"
)

type BaseHandler interface {
	Tunnel(rConn net.Conn, addr *net.UDPAddr) error
}

func MarshalUdpPackage(pkg Com.DataPackage) []byte {
	/*
			sock5 udp data
		   +----+------+--------+----------+----------+------+
		   |RSV | FRAG |  ATYP  | DST.ADDR | DST.PORT | DATA |
		   +----+------+--------+------+----------+----------+
		   | 1  |  0   |    1   | Variable | Variable | Data |
		   +----+------+--------+----------+----------+------+
	*/
	// message
	buf := Com.MarshalPackage(pkg, "udp")
	logger.Debugf("make package is %v", buf)
	return buf
}


func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
