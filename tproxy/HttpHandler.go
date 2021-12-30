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

// tcp handler create handle bind lConn conn and rConn conn
type HttpHandler struct {
	handlerPrv
}

func NewHttpHandler(scope define.Scope, key HandlerKey, proxy config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) *HttpHandler {
	// create new handler
	handler := &HttpHandler{
		handlerPrv: createHandlerPrv(HTTP, scope, key, proxy, lAddr, rAddr, lConn),
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
	//tcpAddr, ok := handler.rAddr.(*net.TCPAddr)
	//if !ok {
	//	logger.Warning("[http] tunnel addr type is not tcp")
	//	return errors.New("type is not tcp")
	//}
	// auth
	auth := auth{
		user:     handler.proxy.UserName,
		password: handler.proxy.Password,
	}
	// create http head
	req := &http.Request{
		Method: http.MethodConnect,
		Host:   handler.rAddr.String(),
		URL: &url.URL{
			Host: handler.rAddr.String(),
		},
		Header: http.Header{},
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
