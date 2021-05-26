package TProxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	config "github.com/ArisAachen/deepin-network-proxy/config"
	define "github.com/ArisAachen/deepin-network-proxy/define"
)


// not use now
// write for env proxy

type HttpHandlerEProxy struct {
	handlerPrv
}

func NewHttpHandlerEProxy(scope define.Scope, key HandlerKey, proxy config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) *HttpHandlerEProxy {
	handler := &HttpHandlerEProxy{
		handlerPrv: createHandlerPrv(HTTP, scope, key, proxy, lAddr, rAddr, lConn),
	}
	handler.saveParent(handler)
	return handler
}

func (handler *HttpHandlerEProxy) Tunnel() error {
	br := bufio.NewReader(handler.lConn)
	lReq, err := http.ReadRequest(br)
	if err != nil {
		logger.Warning(err)
		return err
	}

	// dial proxy server
	rConn, err := handler.dialProxy()
	if err != nil {
		logger.Warningf("[http] failed to dial proxy server, err: %v", err)
		return err
	}
	// auth
	auth := auth{
		user:     handler.proxy.UserName,
		password: handler.proxy.Password,
	}
	if lReq.Method == http.MethodConnect {
		_, err = handler.lConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			logger.Warningf("[http] write 200 failed, err: %v", err)
			return err
		}
	}

	// create http head
	req := &http.Request{
		Method: http.MethodConnect,
		Host:   lReq.Host,
		URL: &url.URL{
			Host: lReq.Host,
		},
		Header: http.Header{
		},
	}
	// check if need auth
	if auth.user != "" && auth.password != "" {
		authMsg := auth.user + ":" + auth.password
		req.Header.Add("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(authMsg)))
	}

	// send connect request to rConn to create tunnel
	logger.Infof("[http] req is %v", req)
	err = req.Write(rConn)
	if err != nil {
		logger.Warningf("[http] write http tunnel request failed, err: %v", err)
		return err
	}
	logger.Info("[http] write req success")
	// read response
	reader := bufio.NewReader(rConn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		logger.Warningf("[http] read response failed, err: %v", err)
		return err
	} else {
		logger.Info("[http] read response success")
	}
	logger.Debug(resp.Status)
	// close body
	defer resp.Body.Close()
	// check if connect success
	if resp.StatusCode != 200 {
		return fmt.Errorf("proxy response error, status code: %v, message: %s",
			resp.StatusCode, resp.Status)
	}
	logger.Infof("[http] proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.lAddr.String(), rConn.RemoteAddr(), handler.rAddr.String())
	// save rConn handler
	handler.rConn = rConn
	return nil
}
