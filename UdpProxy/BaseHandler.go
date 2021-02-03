package UdpProxy

import (
	"encoding/binary"
	"net"
	"pkg.deepin.io/lib/log"
)

type BaseHandler interface {
	Tunnel(rConn net.Conn, addr *net.UDPAddr) error
}

type UdpPackage struct {
	Addr net.UDPAddr
	Data []byte
}

func MarshalUdpPackage(pkg UdpPackage) []byte {
	/*
			sock5 udp data
		   +----+------+--------+----------+----------+------+
		   |RSV | FRAG |  ATYP  | DST.ADDR | DST.PORT | DATA |
		   +----+------+--------+------+----------+----------+
		   | 1  |  0   |    1   | Variable | Variable | Data |
		   +----+------+--------+----------+----------+------+
	*/
	// message
	addr := pkg.Addr
	data := pkg.Data

	// udp message protocol
	buf := make([]byte, 4)
	buf[0] = 0
	buf[1] = 0
	buf[2] = 0
	if addr.IP.To4() != nil {
		buf[3] = 1
		buf = append(buf, addr.IP.To4()...)
	} else if addr.IP.To16() != nil {
		buf[3] = 1
		buf = append(buf, addr.IP.To16()...)
	} else {
		buf[3] = 3
		buf = append(buf, addr.IP...)
	}
	// convert port 2 byte
	portU := uint16(addr.Port)
	if portU == 0 {
		portU = 80
	}
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, portU)
	buf = append(buf, port...)
	// add data
	buf = append(buf, data...)
	logger.Debugf("make package is %v", buf)
	return buf
}

func UnMarshalUdpPackage(msg []byte) UdpPackage {
	addr := msg[4:8]
	port := binary.BigEndian.Uint16(msg[8:10])
	data := msg[10:]

	return UdpPackage{
		Addr: net.UDPAddr{
			IP:   addr[:],
			Port: int(port),
		},
		Data: data,
	}
}

func MarshalPackage(data []byte, addr net.UDPAddr) []byte {
	/*
			sock5 udp data
		   +----+------+--------+----------+----------+------+
		   |RSV | FRAG |  ATYP  | DST.ADDR | DST.PORT | DATA |
		   +----+------+--------+------+----------+----------+
		   | 1  |  0   |    1   | Variable | Variable | Data |
		   +----+------+--------+----------+----------+------+
	*/
	buf := make([]byte, 4)
	buf[0] = 0
	buf[1] = 0
	buf[2] = 0
	if addr.IP.To4() != nil {
		buf[3] = 1
		buf = append(buf, addr.IP.To4()...)
	} else if addr.IP.To16() != nil {
		buf[3] = 1
		buf = append(buf, addr.IP.To16()...)
	} else {
		buf[3] = 3
		buf = append(buf, addr.IP...)
	}
	// convert port 2 byte
	portU := uint16(addr.Port)
	if portU == 0 {
		portU = 80
	}
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, portU)
	buf = append(buf, port...)
	// add data
	buf = append(buf, data...)
	logger.Debugf("make package is %v", buf)
	return buf
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
