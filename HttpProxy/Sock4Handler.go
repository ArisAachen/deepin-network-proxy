package HttpProxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"pkg.deepin.io/lib/log"
	"strconv"
	"time"

	"github.com/proxyTest/Com"
	"github.com/proxyTest/Config"
)

type Sock4Handler struct {
	localHandler  net.Conn
	remoteHandler net.Conn
}

func NewSock4Handler(local net.Conn, proxy Config.Proxy) *Sock4Handler {
	logger.SetLogLevel(log.LevelDebug)
	handler := &Sock4Handler{
		localHandler: local,
	}
	err := handler.invokeProxy(local, proxy)
	if err != nil {
		return nil
	}
	return handler
}

// try to invoke proxy
func (handler *Sock4Handler) invokeProxy(local net.Conn, proxy Config.Proxy) error {
	// get remote addr
	tcpCon, ok := local.(*net.TCPConn)
	if !ok {
		logger.Warningf("local conn is not tcp conn")
		return errors.New("local conn is not tcp conn")
	}
	tcpAddr, err := Com.GetTcpRemoteAddr(tcpCon)
	if err != nil {
		logger.Warningf("get tcp remote addr failed, err: %v", err)
		return nil
	}
	logger.Debugf("remote addr is %v", tcpAddr.String())
	// dial remote server
	port := strconv.Itoa(proxy.Port)
	if port == "" {
		port = "80"
	}
	proxyAddr := proxy.Server + ":" + strconv.Itoa(proxy.Port)
	rConn, err := net.DialTimeout("tcp", proxyAddr, 3*time.Second)
	if err != nil {
		logger.Warningf("connect to remote failed, err: %v", err)
		return err
	}
	// create tunnel
	err = handler.tunnel(rConn, auth{user: proxy.UserName, password: proxy.Password}, tcpAddr)
	if err != nil {
		logger.Warningf("create tunnel failed, err: %v", err)
		return err
	}
	// add remote handler
	handler.remoteHandler = rConn
	return nil
}

func (handler *Sock4Handler) tunnel(rConn net.Conn, auth auth, addr *net.TCPAddr) error {
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
	port := uint16(addr.Port)
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
	buf = append(buf, addr.IP...)
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
		handler.localHandler.RemoteAddr(), rConn.RemoteAddr(), addr.String())
	return nil
}

// communicate
func (handler *Sock4Handler) Communicate() {
	// local to remote
	go func() {
		logger.Debug("copy local -> remote")
		_, err := io.Copy(handler.localHandler, handler.remoteHandler)
		if err != nil {
			logger.Debugf("local to remote closed")
			// ignore close failed
			err = handler.localHandler.Close()
			err = handler.remoteHandler.Close()
		}
	}()
	// remote to local
	go func() {
		logger.Debugf("copy remote -> local")
		_, err := io.Copy(handler.remoteHandler, handler.localHandler)
		if err != nil {
			logger.Debugf("remote to local closed")
			// ignore close failed
			err = handler.localHandler.Close()
			err = handler.remoteHandler.Close()
		}
	}()
}
