package UdpProxy

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"pkg.deepin.io/lib/log"
	"strconv"
	"syscall"
	"unsafe"
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

// set socket transparent
func SetSockTRN() {

}

func GetUdpAddr(con net.UDPConn) {
	rawCon, err := con.SyscallConn()
	if err != nil {
		logger.Fatal(err)
	}
	err = rawCon.Control(func(fd uintptr) {
		buf := make([]byte, 65535)
		sockLen := uint32(len(buf))
		s1, s2, err := unix.Syscall6(unix.SYS_GETSOCKOPT, fd, unix.IPPROTO_IP, unix.IP_OPTIONS, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&sockLen)), 0)
		if err != syscall.Errno(0) {
			logger.Fatal(err)
		}
		logger.Debugf("s1:%v s2:%v buf", s1, s2)
	})

}

func DialUDP(network string, lAddr *net.UDPAddr, rAddr *net.UDPAddr) (*net.UDPConn, error) {
	rSockAddr, err := udpAddrToSockAddr(rAddr)
	if err != nil {
		return nil, err
	}

	lSockAddr, err := udpAddrToSockAddr(lAddr)
	if err != nil {
		return nil, err
	}

	fd, err := syscall.Socket(udpAddrFamily(network, lAddr, rAddr), syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_IP, syscall.IP_TRANSPARENT, 1); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = syscall.Bind(fd, lSockAddr); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if err = syscall.Connect(fd, rSockAddr); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	fdFile := os.NewFile(uintptr(fd), fmt.Sprintf("net-udp-dial-%s", rAddr.String()))
	defer fdFile.Close()

	c, err := net.FileConn(fdFile)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	return c.(*net.UDPConn), nil
}

func udpAddrToSockAddr(addr *net.UDPAddr) (syscall.Sockaddr, error) {
	switch {
	case addr.IP.To4() != nil:
		ip := [4]byte{}
		copy(ip[:], addr.IP.To4())

		return &syscall.SockaddrInet4{Addr: ip, Port: addr.Port}, nil

	default:
		ip := [16]byte{}
		copy(ip[:], addr.IP.To16())

		zoneID, err := strconv.ParseUint(addr.Zone, 10, 32)
		if err != nil {
			zoneID = 0
		}

		return &syscall.SockaddrInet6{Addr: ip, Port: addr.Port, ZoneId: uint32(zoneID)}, nil
	}
}

func udpAddrFamily(net string, lAddr, rAddr *net.UDPAddr) int {
	switch net[len(net)-1] {
	case '4':
		return syscall.AF_INET
	case '6':
		return syscall.AF_INET6
	}

	if (lAddr == nil || lAddr.IP.To4() != nil) && (rAddr == nil || lAddr.IP.To4() != nil) {
		return syscall.AF_INET
	}
	return syscall.AF_INET6
}
