package TProxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/DeepinProxy/Config"
	"pkg.deepin.io/lib/log"
)

type TcpSock5Handler struct {
	localHandler  net.Conn
	remoteHandler net.Conn
	proxy         Config.Proxy
}

func NewTcpSock5Handler(local net.Conn, proxy Config.Proxy) *TcpSock5Handler {
	logger.SetLogLevel(log.LevelDebug)
	handler := &TcpSock5Handler{
		localHandler: local,
		proxy:        proxy,
	}
	return handler
}

// create tunnel between proxy and server
func (handler *TcpSock5Handler) Tunnel(rConn net.Conn, addr net.Addr) error {
	// check type
	tcpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		logger.Warning("[tcp] tunnel addr type is not udp")
		return errors.New("type is not udp")
	}
	// auth message
	auth := auth{
		user:     handler.proxy.UserName,
		password: handler.proxy.Password,
	}
	/*
	    sock5 client hand shake request
	  +----+----------+----------+
	  |VER | NMETHODS | METHODS  |
	  +----+----------+----------+
	  | 1  |    1     | 1 to 255 |
	  +----+----------+----------+
	*/
	// sock5 proto
	// buffer := new(bytes.Buffer)
	buf := make([]byte, 3)
	buf[0] = 5
	buf[1] = 1
	buf[2] = 0
	if auth.user != "" && auth.password != "" {
		buf[1] = 2
		buf = append(buf, byte(2))
	}
	// sock5 hand shake
	_, err := rConn.Write(buf)
	if err != nil {
		logger.Warningf("sock5 hand shake request failed, err: %v", err)
		return err
	}
	/*
		sock5 server hand shake response
		+----+--------+
		|VER | METHOD |
		+----+--------+
		| 1  |   1    |
		+----+--------+
	*/
	_, err = rConn.Read(buf)
	if err != nil {
		logger.Warningf("sock5 hand shake response failed, err: %v", err)
		return err
	}
	logger.Debugf("sock5 hand shake response success message auth method: %v", buf[1])
	if buf[0] != 5 || (buf[1] != 0 && buf[1] != 2) {
		return fmt.Errorf("sock5 proto is invalid, sock type: %v, method: %v", buf[0], buf[1])
	}
	// check if server need auth
	if buf[1] == 2 {
		/*
		    sock5 auth request
		  +----+------+----------+------+----------+
		  |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
		  +----+------+----------+------+----------+
		  | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
		  +----+------+----------+------+----------+
		*/
		buf = make([]byte, 1)
		buf[0] = 1
		buf = append(buf, byte(len(auth.user)))
		buf = append(buf, []byte(auth.user)...)
		buf = append(buf, byte(len(auth.password)))
		buf = append(buf, []byte(auth.password)...)
		// write auth message to writer
		_, err = rConn.Write(buf)
		if err != nil {
			logger.Warningf("sock5 auth request failed, err: %v", err)
			return err
		}
		buf = make([]byte, 32)
		_, err = rConn.Read(buf)
		if err != nil {
			logger.Warningf("sock5 auth response failed, err: %v", err)
			return err
		}
		// RFC1929 user/pass auth should return 1, but some sock5 return 5
		if buf[0] != 5 && buf[0] != 1 {
			logger.Warningf("sock5 auth response incorrect code, code: %v", buf[0])
			return fmt.Errorf("incorrect sock5 auth response, code: %v", buf[0])
		}
		logger.Debugf("sock5 auth success, code: %v", buf[0])
	}
	/*
			sock5 connect request
		   +----+-----+-------+------+----------+----------+
		   |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
		   +----+-----+-------+------+----------+----------+
		   | 1  |  1  | X'00' |  1   | Variable |    2     |
		   +----+-----+-------+------+----------+----------+
	*/
	// start create tunnel
	buf = make([]byte, 4)
	buf[0] = 5
	buf[1] = 1 // connect
	buf[2] = 0 // reserved
	// add tcpAddr
	if tcpAddr.IP.To4() != nil {
		buf[3] = 1
		buf = append(buf, tcpAddr.IP.To4()...)
	} else if tcpAddr.IP.To16() != nil {
		buf[3] = 4
		buf = append(buf, tcpAddr.IP.To16()...)
	} else {
		buf[3] = 3
		buf = append(buf, tcpAddr.IP...)
	}
	// convert port 2 byte
	portU := uint16(tcpAddr.Port)
	if portU == 0 {
		portU = 80
	}
	var port []byte
	binary.BigEndian.PutUint16(port, portU)
	buf = append(buf, port...)
	// request proxy connect remote server
	logger.Debugf("sock5 send connect request, buf: %v", buf)
	_, err = rConn.Write(buf)
	if err != nil {
		logger.Warningf("sock5 send connect request failed, err: %v", err)
		return err
	}
	logger.Debugf("sock5 request successfully")
	buf = make([]byte, 16)
	_, err = rConn.Read(buf)
	if err != nil {
		logger.Warningf("sock5 connect response failed, err: %v", err)
		return err
	}
	logger.Debugf("sock5 response successfully, buf: %v", buf)
	if buf[0] != 5 || buf[1] != 0 {
		logger.Warningf("sock5 connect response failed, version: %v, code: %v", buf[0], buf[1])
		return fmt.Errorf("incorrect sock5 connect reponse, version: %v, code: %v", buf[0], buf[1])
	}
	logger.Debugf("sock5 proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.localHandler.RemoteAddr(), rConn.RemoteAddr(), tcpAddr.String())
	// save remote handler
	handler.remoteHandler = rConn
	return nil
}
