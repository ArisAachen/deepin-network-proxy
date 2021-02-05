package TProxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/DeepinProxy/Config"
)

type Sock4Handler struct {
	localHandler  net.Conn
	remoteHandler net.Conn
	proxy         Config.Proxy
}

func NewSock4Handler(local net.Conn, proxy Config.Proxy) *Sock4Handler {
	handler := &Sock4Handler{
		localHandler: local,
		proxy:        proxy,
	}
	return handler
}

func (handler *Sock4Handler) Tunnel(rConn net.Conn, addr net.Addr) error {
	tcpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		logger.Warning("[tcp] tunnel addr type is not udp")
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
	err := binary.Write(writer, binary.LittleEndian, port)
	if err != nil {
		logger.Warningf("sock4 convert port failed, err: %v", err)
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
	// request proxy connect remote server
	logger.Debugf("sock4 send connect request, buf: %v", buf)
	_, err = rConn.Write(buf)
	if err != nil {
		logger.Warningf("sock4 send connect request failed, err: %v", err)
		return err
	}
	buf = make([]byte, 32)
	_, err = rConn.Read(buf)
	if err != nil {
		logger.Warningf("sock4 connect response failed, err: %v", err)
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
		logger.Warningf("sock4 proto is invalid, sock type: %v, code: %v", buf[0], buf[1])
		return fmt.Errorf("sock4 proto is invalid, sock type: %v, code: %v", buf[0], buf[1])
	}
	logger.Debugf("sock4 proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.localHandler.RemoteAddr(), rConn.RemoteAddr(), tcpAddr.String())
	// save remote handler
	handler.remoteHandler = rConn
	return nil
}
