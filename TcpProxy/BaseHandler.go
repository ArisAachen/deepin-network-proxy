package TcpProxy

import (
	"io"
	"net"
	"strconv"
	"time"

	"github.com/DeepinProxy/Config"
	"pkg.deepin.io/lib/log"
)

var logger *log.Logger

type BaseHandler interface {
	Tunnel(rConn net.Conn, addr *net.TCPAddr) error
}

// dial proxy
func DialProxy(proxy Config.Proxy) (net.Conn, error) {
	port := strconv.Itoa(proxy.Port)
	if port == "" {
		port = "80"
	}
	proxyAddr := proxy.Server + ":" + strconv.Itoa(proxy.Port)
	return net.DialTimeout("tcp", proxyAddr, 3*time.Second)
}

// conn communicate
func Communicate(local net.Conn, remote net.Conn) {
	// local to remote
	go func() {
		logger.Debug("copy local -> remote")
		_, err := io.Copy(local, remote)
		if err != nil {
			logger.Debugf("local to remote closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
	// remote to local
	go func() {
		logger.Debugf("copy remote -> local")
		_, err := io.Copy(remote, local)
		if err != nil {
			logger.Debugf("remote to local closed")
			// ignore close failed
			err = local.Close()
			err = remote.Close()
		}
	}()
}

func NewHandler(local net.Conn, proxy Config.Proxy, proto string) BaseHandler {
	// search proto
	switch proto {
	case "http":
		return NewHttpHandler(local, proxy)
	case "sock4":
		return NewSock4Handler(local, proxy)
	case "sock5":
		return NewSock5Handler(local, proxy)
	default:
		logger.Warningf("unknown proto type: %v", proto)
	}
	return nil
}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
