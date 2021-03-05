package TProxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/DeepinProxy/Config"
	define "github.com/DeepinProxy/Define"
)

type Sock4Handler struct {
	handlerPrv
}

func NewSock4Handler(scope define.Scope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) *Sock4Handler {
	// create new handler
	handler := &Sock4Handler{
		handlerPrv: createHandlerPrv(SOCK4, scope, key, proxy, lAddr, rAddr, lConn),
	}
	// add self to private parent
	handler.saveParent(handler)
	return handler
}

func (handler *Sock4Handler) Tunnel() error {
	// dial proxy server
	rConn, err := handler.dialProxy()
	if err != nil {
		logger.Warningf("[sock4] failed to dial proxy server, err: %v", err)
		return err
	}
	// check type
	tcpAddr, ok := handler.rAddr.(*net.TCPAddr)
	if !ok {
		logger.Warning("[sock4] tunnel addr type is not udp")
		return errors.New("type is not udp")
	}
	// sock4 dont support password auth
	auth := auth{
		user: handler.proxy.UserName,
	}
	/*
					sock4 connect request
				+----+----+----+----+----+----+----+----+----+----+....+----+
				| VN | CD | DSTPORT |      DSTIP        | USERID       |NULL|
				+----+----+----+----+----+----+----+----+----+----+....+----+
		           1    1      2              4           variable       1
	*/
	buf := make([]byte, 2)
	buf[0] = 4 // sock version
	buf[1] = 1 // connect command
	// convert port 2 byte
	writer := new(bytes.Buffer)
	port := uint16(tcpAddr.Port)
	if port == 0 {
		port = 80
	}
	err = binary.Write(writer, binary.LittleEndian, port)
	if err != nil {
		logger.Warningf("[sock4] convert port failed, err: %v", err)
		return err
	}
	portBy := writer.Bytes()[0:2]
	buf = append(buf, portBy...)
	// add ip and user
	buf = append(buf, tcpAddr.IP...)
	if auth.user != "" {
		buf = append(buf, []byte(auth.user)...)
	} else {
		buf = append(buf, uint8(0))
	}
	// request proxy connect rConn server
	logger.Debugf("[sock4] send connect request, buf: %v", buf)
	_, err = rConn.Write(buf)
	if err != nil {
		logger.Warningf("[sock4] send connect request failed, err: %v", err)
		return err
	}
	buf = make([]byte, 32)
	_, err = rConn.Read(buf)
	if err != nil {
		logger.Warningf("[sock4] connect response failed, err: %v", err)
		return err
	}
	/*
					sock4 server response
				+----+----+----+----+----+----+----+----+
				| VN | CD | DSTPORT |      DSTIP        |
				+----+----+----+----+----+----+----+----+
		          1    1      2              4

	*/
	// 0   0x5A
	if buf[0] != 0 || buf[1] != 90 {
		logger.Warningf("[sock4] proto is invalid, sock type: %v, code: %v", buf[0], buf[1])
		return fmt.Errorf("sock4 proto is invalid, sock type: %v, code: %v", buf[0], buf[1])
	}
	logger.Debugf("[sock4] proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.lConn.RemoteAddr(), rConn.RemoteAddr(), tcpAddr.String())
	// save rConn handler
	handler.lConn = rConn
	return nil
}
