package Com

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"syscall"

	"golang.org/x/sys/unix"
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

// set socket transparent
func SetSockOptTrn(fd int) error {
	soTyp, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TYPE)
	if err != nil {
		return err
	}
	// check if type match
	if soTyp != syscall.SOCK_STREAM && soTyp != syscall.SOCK_DGRAM {
		return errors.New("sock type is not tcp and udp")
	}
	// set reuse addr
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}
	// set ip transparent
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.IP_TRANSPARENT, 1); err != nil {
		return err
	}
	return nil
}

// parse origin remote addr msg from msg_hdr
func ParseRemoteAddrFromMsgHdr(buf []byte) (net.Addr, error) {
	if buf == nil {
		return nil, errors.New("parse buf is nil")
	}
	// parse control message
	msgSl, err := syscall.ParseSocketControlMessage(buf)
	if err != nil {
		return nil, err
	}
	var addr *net.TCPAddr
	// tcp and udp addr is the same struct, use tcp to represent all
	for _, msg := range msgSl {
		// use t_proxy and ip route, msg_hdr address is marked as sol_ip type
		if msg.Header.Level == syscall.SOL_IP && msg.Header.Type == syscall.IP_RECVORIGDSTADDR {
			addr = &net.TCPAddr{
				IP:   msg.Data[4:8],
				Port: int(binary.BigEndian.Uint16(msg.Data[2:4])),
			}
		} else if msg.Header.Level == syscall.SOL_IPV6 && msg.Header.Type == syscall.IP_RECVORIGDSTADDR {
			addr = &net.TCPAddr{
				IP:   msg.Data[8:24],
				Port: int(binary.BigEndian.Uint16(msg.Data[2:4])),
			}
		}
	}
	// check if addr is nil
	if addr == nil {
		err = errors.New("sol_ip type is not found int msg_hdr")
	}
	return addr, err
}

// mega dial try to transparent connect, privilege should be needed
func MegaDial(network string, lAddr net.Addr, rAddr net.Addr) (net.Conn, error) {
	// check if is the same type, udp addr can not dial tcp addr
	if reflect.TypeOf(lAddr) != reflect.TypeOf(rAddr) {
		return nil, errors.New("dial local addr is not match with remote addr")
	}
	// get domain
	var domain int
	var ip net.IP = reflect.ValueOf(lAddr).FieldByName("IP").Bytes()
	if ip.To4() != nil {
		domain = syscall.AF_INET
	} else if ip.To16() != nil {
		domain = syscall.AF_INET6
	} else {
		return nil, errors.New("local ip is incorrect")
	}
	// get typ
	var typ int
	if network == "tcp" {
		typ = syscall.SOCK_STREAM
	} else if network == "udp" {
		typ = syscall.SOCK_DGRAM
	}
	fd, err := syscall.Socket(domain, typ, 0)
	if err != nil {
		return nil, err
	}
	// set transparent
	if err = SetSockOptTrn(fd); err != nil {
		return nil, err
	}
	// convert addr
	lSockAddr, err := convertAddrToSockAddr(lAddr)
	if err != nil {
		return nil, err
	}
	rSockAddr, err := convertAddrToSockAddr(rAddr)
	if err != nil {
		return nil, err
	}
	// bind fake addr
	if err = syscall.Bind(fd, lSockAddr); err != nil {
		return nil, err
	}
	// bind addr
	if err = syscall.Connect(fd, rSockAddr); err != nil {
		return nil, err
	}
	// create new file
	file := os.NewFile(uintptr(fd), fmt.Sprintf("udp_handler_%v", fd))
	if file == nil {
		return nil, errors.New("create new file is nil")
	}
	// create file conn
	conn, err := net.FileConn(file)
	if err != nil {
		return nil, err
	}
	// debug message
	return conn, nil
}

// convert addr to sock addr
func convertAddrToSockAddr(addr net.Addr) (syscall.Sockaddr, error) {
	// check if addr can convert to udp addr and tcp addr, if not return as error
	if !reflect.TypeOf(addr).ConvertibleTo(reflect.TypeOf(net.UDPAddr{})) &&
		!reflect.TypeOf(addr).ConvertibleTo(reflect.TypeOf(net.TCPAddr{})) {
		return nil, errors.New("addr typ is not tcp addr or udp addr")
	}
	// convert net addr to sock_addr
	value := reflect.ValueOf(addr)
	var ip net.IP = value.FieldByName("IP").Bytes()
	port := value.FieldByName("Port").Int()
	if port == 0 {
		port = 80
	}
	if ip.To4() != nil {
		inet4 := &syscall.SockaddrInet4{
			Port: int(port),
		}
		copy(inet4.Addr[:], ip.To4())
		return inet4, nil
	} else if ip.To16() != nil {
		inet6 := &syscall.SockaddrInet6{
			Port: int(port),
		}
		copy(inet6.Addr[:], ip.To16())
		return inet6, nil
	}
	return nil, errors.New("ip is not ipv4 or ipv6")
}
