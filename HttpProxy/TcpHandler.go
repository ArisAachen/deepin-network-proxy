package HttpProxy

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/proxyTest/Com"
	"github.com/proxyTest/Config"
	"io"
	"net"
	"net/http"
	"net/url"
	"pkg.deepin.io/lib/log"
	"strconv"
	"time"
)

// tcp handler create handle bind local conn and remote conn
type TcpHandler struct {
	localHandler  net.Conn
	remoteHandler net.Conn
}

// auth message
type auth struct {
	user     string
	password string
}

var logger = log.NewLogger("daemon/session/proxy")

func NewTcpHandler(local net.Conn, proxy Config.Proxy) *TcpHandler {
	logger.SetLogLevel(log.LevelDebug)
	// create handler
	handler := &TcpHandler{
		localHandler: local,
	}
	// try to invoke proxy
	err := handler.invokeProxy(local, proxy)
	if err != nil {
		return nil
	}
	return handler
}

// try to invoke proxy
func (handler *TcpHandler) invokeProxy(local net.Conn, proxy Config.Proxy) error {
	// get remote addr
	tcpCon, ok := local.(*net.TCPConn)
	if !ok {
		logger.Warningf("local conn is not tcp conn")
		return errors.New("local conn is not tcp conn")
	}
	tcpAddr, err := Com.GetTcpRemoteAddr(tcpCon)
	logger.Debugf("remote addr is %v", tcpAddr.String())
	if err != nil {
		logger.Warningf("get tcp remote addr failed, err: %v", err)
		return nil
	}
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

// create tunnel between proxy and server
func (handler *TcpHandler) tunnel(rConn net.Conn, auth auth, addr *net.TCPAddr) error {
	// create http head
	req := &http.Request{
		Method: http.MethodConnect,
		Host:   addr.String(),
		URL: &url.URL{
			Host: addr.String(),
		},
		Header: http.Header{
			//"Proxy-Connection": []string{"Keep-Alive"},
		},
	}
	// check if need auth
	if auth.user != "" && auth.password != "" {
		authMsg := auth.user + ":" + auth.password
		req.Header.Add("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(authMsg)))
	}
	// send connect request to remote to create tunnel
	err := req.Write(rConn)
	if err != nil {
		return err
	}
	// read response
	reader := bufio.NewReader(rConn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return err
	}
	logger.Debug(resp.Status)
	// close body
	defer resp.Body.Close()
	// check if connect success
	if resp.StatusCode != 200 {
		return fmt.Errorf("proxy response error, status code: %v, message: %s",
			resp.StatusCode, resp.Status)
	}
	logger.Debugf("tunnel create success, [%s] -> [%s] -> [%s]",
		handler.localHandler.RemoteAddr(), rConn.RemoteAddr(), addr.String())
	return nil
}

// communicate
func (handler *TcpHandler) Communicate() {
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
