package TcpProxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/DeepinProxy/Config"
)

// tcp handler create handle bind local conn and remote conn
type HttpHandler struct {
	localHandler  net.Conn
	remoteHandler net.Conn
	proxy         Config.Proxy
}

// auth message
type auth struct {
	user     string
	password string
}

func NewHttpHandler(local net.Conn, proxy Config.Proxy) *HttpHandler {
	// create handler
	handler := &HttpHandler{
		localHandler: local,
		proxy:        proxy,
	}
	return handler
}

// create tunnel between proxy and server
func (handler *HttpHandler) Tunnel(rConn net.Conn, addr *net.TCPAddr) error {
	// auth
	auth := auth{
		user:     handler.proxy.UserName,
		password: handler.proxy.Password,
	}
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
	logger.Debugf("http proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.localHandler.RemoteAddr(), rConn.RemoteAddr(), addr.String())
	// save remote handler
	handler.remoteHandler = rConn
	return nil
}
