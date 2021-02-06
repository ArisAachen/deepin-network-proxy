package TProxy

import (
	"errors"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/DeepinProxy/Config"
)

// handler private, data of handler

type handlerPrv struct {
	// config message
	scope ProxyScope
	proxy Config.Proxy

	// connection
	lAddr net.Addr
	rAddr net.Addr
	lConn net.Conn
	rConn net.Conn

	// map key
	parent BaseHandler
	key    HandlerKey
	mgr    *HandlerMgr
}

// new handler private
func newHandlerPrv(scope ProxyScope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr) *handlerPrv {
	return &handlerPrv{
		scope: scope,
		lAddr: lAddr,
		rAddr: rAddr,
		key:   key,
		proxy: proxy,
	}
}

// save parent
func (pr *handlerPrv) saveParent(parent BaseHandler) {
	pr.parent = parent
}

// add private to manager and save manager
func (pr *handlerPrv) AddMgr(mgr *HandlerMgr) {
	// check parent
	if pr.parent == nil {
		logger.Warningf("handler private has no parent")
	}
	// add private manager
	pr.mgr = mgr
	// add parent to manager
	mgr.AddHandler(pr.scope, pr.key, pr.parent)
}

// tcp connect to remote server
func (pr *handlerPrv) dialProxy() (net.Conn, error) {
	proxy := pr.proxy
	if proxy.Port == 0 {
		proxy.Port = 80
	}
	server := proxy.Server + ":" + strconv.Itoa(proxy.Port)
	conn, err := net.DialTimeout("tcp", server, 3*time.Second)
	if err != nil {
		logger.Warningf("dial proxy server failed, err: %v", err)
		return nil, err
	}
	logger.Debugf("dial proxy server success, [%s] -> [%s]", conn.LocalAddr(), conn.RemoteAddr())
	return conn, nil
}

// read and write

func (pr *handlerPrv) WriteRemote(buf []byte) error {
	if pr.rConn == nil {
		return errors.New("remote handler is nil")
	}
	_, err := pr.rConn.Write(buf)
	if err != nil {
		logger.Warningf("write remote failed, err: %v", err)
		return err
	}
	return nil
}

func (pr *handlerPrv) WriteLocal(buf []byte) error {
	if pr.lConn == nil {
		return errors.New("remote handler is nil")
	}
	_, err := pr.lConn.Write(buf)
	if err != nil {
		logger.Warningf("write remote failed, err: %v", err)
		return err
	}
	return nil
}

func (pr *handlerPrv) ReadRemote(buf []byte) error {
	if pr.rConn == nil {
		return errors.New("remote handler is nil")
	}
	_, err := pr.rConn.Read(buf)
	if err != nil {
		logger.Warningf("write remote failed, err: %v", err)
		return err
	}
	return nil
}

func (pr *handlerPrv) ReadLocal(buf []byte) error {
	if pr.lConn == nil {
		return errors.New("remote handler is nil")
	}
	_, err := pr.lConn.Read(buf)
	if err != nil {
		logger.Warningf("write remote failed, err: %v", err)
		return err
	}
	return nil
}

// communicate lConn and rConn
func (pr *handlerPrv) Communicate() {
	go func() {
		logger.Debugf("begin copy data, local [%s] -> remote [%s]", pr.lAddr.String(), pr.rAddr.String())
		_, err := io.Copy(pr.lConn, pr.rConn)
		if err != nil {
			logger.Debugf("stop copy data, local [%s] =x= remote [%s], reason: %v", pr.lAddr.String(), pr.rAddr.String(), err)
		}
		pr.Remove()
	}()
	go func() {
		logger.Debugf("begin copy data, remote [%s] -> local [%s]", pr.rAddr.String(), pr.lAddr.String())
		_, err := io.Copy(pr.rConn, pr.lConn)
		if err != nil {
			logger.Debugf("stop copy data, remote [%s] =x= local [%s], reason: %v", pr.rAddr.String(), pr.lAddr.String(), err)
		}
		pr.Remove()
	}()
}

// close handler
func (pr *handlerPrv) Close() {
	if pr.lConn != nil {
		_ = pr.lConn.Close()
	}
	if pr.rConn != nil {
		_ = pr.rConn.Close()
	}
}

// close and delete handler from manager
func (pr *handlerPrv) Remove() {
	pr.mgr.CloseBaseHandler(pr.scope, pr.key)
}
