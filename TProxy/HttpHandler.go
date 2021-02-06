package TProxy

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/DeepinProxy/Config"
)

// tcp handler create handle bind lConn conn and rConn conn
type HttpHandler struct {
	handlerPrv
}

func NewHttpHandler(scope ProxyScope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) *HttpHandler {
	// create new handler
	handler := &HttpHandler{
		handlerPrv: handlerPrv{
			// config
			scope: scope,
			key:   key,
			proxy: proxy,

			// connection
			lAddr: lAddr,
			rAddr: rAddr,
			lConn: lConn,
		},
	}
	// add self to private parent
	handler.saveParent(handler)
	return handler
}

// create tunnel between proxy and server
func (handler *HttpHandler) Tunnel() error {
	// dial proxy server
	rConn, err := handler.dialProxy()
	if err != nil {
		logger.Warningf("[http] failed to dial proxy server, err: %v", err)
		return err
	}
	// check type
	tcpAddr, ok := handler.rAddr.(*net.TCPAddr)
	if !ok {
		logger.Warning("[http] tunnel addr type is not udp")
		return errors.New("type is not udp")
	}
	// auth
	auth := auth{
		user:     handler.proxy.UserName,
		password: handler.proxy.Password,
	}
	// create http head
	req := &http.Request{
		Method: http.MethodConnect,
		Host:   tcpAddr.String(),
		URL: &url.URL{
			Host: tcpAddr.String(),
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
	// send connect request to rConn to create tunnel
	err = req.Write(rConn)
	if err != nil {
		logger.Warningf("[http] write http tunnel request failed, err: %v", err)
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
	logger.Debugf("[http] proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.lAddr.String(), rConn.RemoteAddr(), handler.rAddr.String())
	// save rConn handler
	handler.rConn = rConn
	return nil
}
