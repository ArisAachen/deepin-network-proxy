package main

import (
	"encoding/binary"
	"net"
	"pkg.deepin.io/lib/log"
	"strconv"
	"syscall"
	"time"

	com "github.com/DeepinProxy/Com"
	cfg "github.com/DeepinProxy/Config"
	tcpProxy "github.com/DeepinProxy/TcpProxy"
	udpProxy "github.com/DeepinProxy/UdpProxy"
)

var logger = log.NewLogger("daemon/proxy")

func main() {
	NewUdpProxy()
}

// https://www.kernel.org/doc/Documentation/networking/tproxy.txt
func NewUdpProxy() {
	l, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		logger.Warningf("listen udp package port failed, err: %v", err)
		return
	}
	defer l.Close()

	// ip_transparent
	conn, ok := l.(*net.UDPConn)
	if !ok {
		logger.Fatal(err)
	}

	file, err := conn.File()
	if err != nil {
		logger.Fatal(err)
	}

	//
	if err = syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1); err != nil {
		logger.Fatal(err)
	}

	if err = syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_IP, syscall.IP_RECVORIGDSTADDR, 1); err != nil {
		logger.Fatal(err)
	}

	config := cfg.NewProxyCfg()
	err = config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
	if err != nil {
		logger.Warningf("load config failed, err: %v", err)
		return
	}
	proxy, err := config.GetProxy("global", "sock5")
	if err != nil {
		logger.Warningf("get proxy from config failed, err: %v", err)
		return
	}

	// dial proxy server
	addr := proxy.Server
	port := strconv.Itoa(proxy.Port)
	if port == "" {
		port = "80"
	}

	for {
		buf := make([]byte, 512)
		oob := make([]byte, 1024)
		_, oobn, _, lAddr, err := conn.ReadMsgUDP(buf, oob)
		if err != nil {
			logger.Fatal(err)
		}

		_, err = conn.WriteTo(buf, lAddr)
		if err != nil {
			logger.Fatal(err)
		}

		msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			logger.Fatal(err)
		}

		var udpAddr *net.UDPAddr = nil
		for _, msg := range msgs {
			if msg.Header.Level == syscall.SOL_IP && msg.Header.Type == syscall.IP_RECVORIGDSTADDR {
				ip := net.IP(msg.Data[4:8])
				port := binary.BigEndian.Uint16(msg.Data[2:4])
				logger.Infof("result is %v %v", ip, port)
				udpAddr = &net.UDPAddr{
					IP:   msg.Data[4:8],
					Port: int(msg.Data[2])<<8 + int(msg.Data[3]),
				}
			}
		}

		logger.Infof("recv buf is %v", buf)

		remoteConn, err := net.DialTimeout("tcp", addr+":"+port, 3*time.Second)
		if err != nil {
			logger.Warning(err)
			continue
		}

		handler := udpProxy.NewSock5Handler(lAddr, proxy)
		err = handler.Tunnel(remoteConn, udpAddr)
		if err != nil {
			logger.Warningf("tunnel failed, err: %v", err)
			continue
		}

		buf = udpProxy.MarshalUdpPackage(udpProxy.UdpPackage{Addr: *udpAddr, Data: buf})
		_, err = handler.RemoteHandler.Write(buf)
		if err != nil {
			logger.Fatal(err)
		}

		buf = make([]byte, 512)
		_, err = handler.RemoteHandler.Read(buf)
		if err != nil {
			logger.Fatal(err)
		}
		pkg := udpProxy.UnMarshalUdpPackage(buf)
		_, err = conn.WriteTo(pkg.Data, lAddr)
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func NewTcpProxy(listen string) {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		logger.Warningf("listen tcp port failed, err: %v", err)
		return
	}
	defer l.Close()

	config := cfg.NewProxyCfg()
	err = config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
	if err != nil {
		logger.Warningf("load config failed, err: %v", err)
		return
	}

	proxy, err := config.GetProxy("global", "sock5")
	if err != nil {
		logger.Warningf("get proxy from config failed, err: %v", err)
		return
	}

	for {
		localConn, err := l.Accept()
		if err != nil {
			logger.Warningf("accept socket failed, err: %v", err)
			continue
		}

		// get remote addr
		tcpCon, ok := localConn.(*net.TCPConn)
		if !ok {
			logger.Warningf("accept conn type is not tcp conn %v", err)
			err = localConn.Close()
			continue
		}
		tcpAddr, err := com.GetTcpRemoteAddr(tcpCon)
		if err != nil {
			logger.Warningf("get remote addr failed, err: %v", err)
			err = localConn.Close()
			continue
		}
		// create handler
		handler := tcpProxy.NewHandler(localConn, proxy, "sock5")
		// dial proxy server
		addr := proxy.Server
		port := strconv.Itoa(proxy.Port)
		if port == "" {
			port = "80"
		}
		remoteConn, err := net.DialTimeout("tcp", addr+":"+port, 3*time.Second)
		if err != nil {
			logger.Warningf("dial remote proxy server failed, err: %v", err)
			err = localConn.Close()
			continue
		}
		err = handler.Tunnel(remoteConn, tcpAddr)
		if err != nil {
			logger.Warningf("create tunnel failed, %v", err)
			err = localConn.Close()
			err = remoteConn.Close()
		}
		tcpProxy.Communicate(localConn, remoteConn)
	}
}
