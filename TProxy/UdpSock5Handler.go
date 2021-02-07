package TProxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/DeepinProxy/Com"
	"github.com/DeepinProxy/Config"
	"io"
	"net"
)

type UdpSock5Handler struct {
	handlerPrv
	rTcpConn net.Conn
}

func NewUdpSock5Handler(scope ProxyScope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) *UdpSock5Handler {
	// create new handler
	handler := &UdpSock5Handler{
		handlerPrv: createHandlerPrv(SOCK5UDP, scope, key, proxy, lAddr, rAddr, lConn),
	}
	// add self to private parent
	handler.saveParent(handler)
	return handler
}

// rewrite close
func (handler *UdpSock5Handler) Close() {
	if handler.rTcpConn != nil {
		_ = handler.rTcpConn.Close()
	}
	handler.handlerPrv.Close()
}

// rewrite read remote
func (handler *UdpSock5Handler) Read(buf []byte) (int, error) {
	// check if rConn is nil
	if handler.rConn == nil {
		return 0, errors.New("remote handler is nil")
	}
	data := make([]byte, 512)
	n, err := handler.rConn.Read(data)
	if err != nil {
		logger.Warningf("read remote failed, err: %v", err)
		return n, err
	}
	pkgData := Com.UnMarshalPackage(data)
	copy(buf, pkgData.Data)
	return n, nil
}

// rewrite write remote
func (handler *UdpSock5Handler) Write(buf []byte) (int, error) {
	if handler.rConn == nil {
		return 0, errors.New("remote handler is nil")
	}
	pkgData := Com.DataPackage{
		Addr: handler.rAddr,
		Data: buf,
	}
	_, err := handler.rConn.Write(Com.MarshalPackage(pkgData, "udp"))
	if err != nil {
		return 0, err
	}
	return len(buf), nil
}

// rewrite communication
func (handler *UdpSock5Handler) Communicate() {
	// local -> remote
	go func() {
		logger.Debugf("[%s] begin copy data, local [%s] -> remote [%s]", handler.typ, handler.lAddr.String(), handler.rAddr.String())
		_, err := io.Copy(handler.lConn, handler)
		if err != nil {
			logger.Debugf("[%s] stop copy data, local [%s] -x- remote [%s], reason: %v",
				handler.typ, handler.lAddr.String(), handler.rAddr.String(), err)
		}
		handler.Remove()
	}()

	// remote -> local
	go func() {
		logger.Debugf("[%s] begin copy data, remote [%s] -> local [%s]", handler.typ, handler.lAddr.String(), handler.rAddr.String())
		_, err := io.Copy(handler, handler.lConn)
		if err != nil {
			logger.Debugf("[%s] stop copy data, remote [%s] -x- local [%s], reason: %v",
				handler.typ, handler.lAddr.String(), handler.rAddr.String(), err)
		}
		handler.Remove()
	}()
}

// create tunnel between proxy and server
func (handler *UdpSock5Handler) Tunnel() error {
	// dial proxy server
	rTcpConn, err := handler.dialProxy()
	if err != nil {
		logger.Warningf("[udp] failed to dial proxy server, err: %v", err)
		return err
	}
	// save tcp connection
	handler.rTcpConn = rTcpConn
	// check type
	udpAddr, ok := handler.rAddr.(*net.UDPAddr)
	if !ok {
		logger.Warning("[udp] tunnel addr type is not udp")
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
	_, err = rTcpConn.Write(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 hand shake request failed, err: %v", err)
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
	_, err = rTcpConn.Read(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 hand shake response failed, err: %v", err)
		return err
	}
	logger.Debugf("[udp] sock5 hand shake response success message auth method: %v", buf[1])
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
		_, err = rTcpConn.Write(buf)
		if err != nil {
			logger.Warningf("[udp] sock5 auth request failed, err: %v", err)
			return err
		}
		buf = make([]byte, 32)
		_, err = rTcpConn.Read(buf)
		if err != nil {
			logger.Warningf("[udp] sock5 auth response failed, err: %v", err)
			return err
		}
		// RFC1929 user/pass auth should return 1, but some sock5 return 5
		if buf[0] != 5 && buf[0] != 1 {
			logger.Warningf("[udp] sock5 auth response incorrect code, code: %v", buf[0])
			return fmt.Errorf("incorrect sock5 auth response, code: %v", buf[0])
		}
		logger.Debugf("[udp] sock5 auth success, code: %v", buf[0])
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
	buf[1] = 3 // udp
	buf[2] = 0 // reserved
	// add addr
	if udpAddr.IP.To4() != nil {
		buf[3] = 1
		buf = append(buf, net.IP{127, 0, 0, 0}...)
	} else if udpAddr.IP.To16() != nil {
		buf[3] = 4
		buf = append(buf, udpAddr.IP.To16()...)
	} else {
		buf[3] = 3
		buf = append(buf, udpAddr.IP...)
	}
	// convert port 2 byte
	portU := uint16(udpAddr.Port)
	if portU == 0 {
		portU = 80
	}
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, portU)
	buf = append(buf, port...)
	// request proxy connect rTcpConn server
	logger.Debugf("[udp] sock5 send connect request, buf: %v", buf)
	_, err = rTcpConn.Write(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 send connect request failed, err: %v", err)
		return err
	}
	logger.Debugf("[udp] sock5 request successfully")
	buf = make([]byte, 16)
	_, err = rTcpConn.Read(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 connect response failed, err: %v", err)
		return err
	}
	logger.Debugf("[udp] sock5 response successfully, buf: %v", buf)
	if buf[0] != 5 || buf[1] != 0 {
		logger.Warningf("[udp] sock5 connect response failed, version: %v, code: %v", buf[0], buf[1])
		return fmt.Errorf("[udp] incorrect sock5 connect reponse, version: %v, code: %v", buf[0], buf[1])
	}
	// dial rTcpConn udp server
	udpServer := net.UDPAddr{
		IP:   buf[4:8],
		Port: int(binary.BigEndian.Uint16(buf[8:10])),
	}
	udpConn, err := net.Dial("udp", udpServer.String())
	if err != nil {
		logger.Warningf("[udp] dial rTcpConn udp failed, err: %v", err)
		return err
	}

	logger.Debugf("[udp] sock5 proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.lAddr.String(), udpServer.String(), handler.rAddr.String())
	// save rTcpConn handler
	handler.rConn = udpConn
	return nil
}
