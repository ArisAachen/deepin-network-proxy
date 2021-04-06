package TProxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/DeepinProxy/Config"
	define "github.com/DeepinProxy/Define"
)

type TcpSock5Handler struct {
	handlerPrv
}

func NewTcpSock5Handler(scope define.Scope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) *TcpSock5Handler {
	// create new handler
	handler := &TcpSock5Handler{
		handlerPrv: createHandlerPrv(SOCK5TCP, scope, key, proxy, lAddr, rAddr, lConn),
	}
	// add self to private parent
	handler.saveParent(handler)
	return handler
}

// create tunnel between proxy and server
func (handler *TcpSock5Handler) Tunnel() error {
	// dial proxy server
	rConn, err := handler.dialProxy()
	if err != nil {
		logger.Warningf("[%s] failed to dial proxy server, err: %v", handler.typ, err)
		return err
	}
	// check type
	tcpAddr, ok := handler.rAddr.(*net.TCPAddr)
	if !ok {
		logger.Warningf("[%s] tunnel addr type is not tcp", handler.typ)
		return errors.New("type is not tcp")
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
	_, err = rConn.Write(buf)
	if err != nil {
		logger.Warningf("[%s] hand shake request failed, err: %v", handler.typ, err)
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
		logger.Warningf("[%s] hand shake response failed, err: %v", handler.typ, err)
		return err
	}
	logger.Debugf("[%s] hand shake response success message auth method: %v", handler.typ, buf[1])
	if buf[0] != 5 || (buf[1] != 0 && buf[1] != 2) {
		return fmt.Errorf("sock5 proto is invalid, sock type: %v, method: %v", buf[0], buf[1])
	}
	// check if server need auth
	if buf[1] == 2 {
		logger.Debugf("[%s] proxy need auth, start authenticating...", handler.typ)
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
			logger.Warningf("[%s] auth request failed, err: %v", handler.typ, err)
			return err
		}
		buf = make([]byte, 32)
		_, err = rConn.Read(buf)
		if err != nil {
			logger.Warningf("[%s] auth response failed, err: %v", handler.typ, err)
			return err
		}
		// RFC1929 user/pass auth should return 1, but some sock5 return 5
		if buf[0] != 5 && buf[0] != 1 {
			logger.Warningf("[%s] auth response incorrect code, code: %v", handler.typ, buf[0])
			return fmt.Errorf("incorrect sock5 auth response, code: %v", buf[0])
		}
		logger.Debugf("[%s] auth success, code: %v", handler.typ, buf[0])
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
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, portU)
	buf = append(buf, port...)
	// request proxy connect rConn server
	logger.Debugf("[%s] send connect request, buf: %v", handler.typ, buf)
	_, err = rConn.Write(buf)
	if err != nil {
		logger.Warningf("[%s] send connect request failed, err: %v", handler.typ, err)
		return err
	}
	logger.Debugf("[%s] request successfully", handler.typ)
	buf = make([]byte, 16)
	_, err = rConn.Read(buf)
	if err != nil {
		logger.Warningf("[%s] connect response failed, err: %v", handler.typ, err)
		return err
	}
	logger.Debugf("[%s] response successfully, buf: %v", handler.typ, buf)
	if buf[0] != 5 || buf[1] != 0 {
		logger.Warningf("[%s] connect response failed, version: %v, code: %v", handler.typ, buf[0], buf[1])
		return fmt.Errorf("incorrect sock5 connect reponse, version: %v, code: %v", buf[0], buf[1])
	}
	logger.Debugf("[%s] proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.typ, handler.lAddr.String(), rConn.RemoteAddr(), handler.rAddr.String())
	// save rConn handler
	handler.rConn = rConn
	return nil
}
