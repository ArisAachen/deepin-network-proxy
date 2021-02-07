package TProxy

import (
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/DeepinProxy/Config"
)

// handler private, data of handler

type handlerPrv struct {
	typ ProtoTyp

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

	// delete mark, in case if delete twice, not use this time
	deleted bool
	lock    sync.Mutex
}

// new handler private
func createHandlerPrv(typ ProtoTyp, scope ProxyScope, key HandlerKey, proxy Config.Proxy, lAddr net.Addr, rAddr net.Addr, lConn net.Conn) handlerPrv {
	return handlerPrv{
		// proxy typ
		typ: typ,

		// config
		scope: scope,
		key:   key,
		proxy: proxy,

		// connection
		lAddr: lAddr,
		rAddr: rAddr,
		lConn: lConn,

		// delete mark
		deleted: false,
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
	mgr.AddHandler(pr.scope, pr.typ, pr.key, pr.parent)
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
		logger.Warningf("[%s] dial proxy server failed, err: %v", pr.typ, err)
		return nil, err
	}
	logger.Debugf("[%s] dial proxy server success, local [%s] -> remote [%s]", pr.typ, conn.LocalAddr(), conn.RemoteAddr())
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
		logger.Debugf("[%s] begin copy data, local [%s] -> remote [%s]", pr.typ, pr.lAddr.String(), pr.rAddr.String())
		_, err := io.Copy(pr.lConn, pr.rConn)
		if err != nil {
			logger.Debugf("[%s] stop copy data, local [%s] -x- remote [%s], reason: %v", pr.typ, pr.lAddr.String(), pr.rAddr.String(), err)
		}
		// mark deleted, but not actually deleted at this time, only set a mark
		if pr.isDeleted() {
			return
		}
		pr.setDeleted(true)
		// remove handler from map
		pr.Remove()
	}()
	go func() {
		logger.Debugf("[%s] begin copy data, remote [%s] -> local [%s]", pr.typ, pr.rAddr.String(), pr.lAddr.String())
		_, err := io.Copy(pr.rConn, pr.lConn)
		if err != nil {
			logger.Debugf("[%s] stop copy data, local [%s] -x- remote [%s], reason: %v", pr.typ, pr.rAddr.String(), pr.lAddr.String(), err)
		}
		// mark deleted, but not actually deleted at this time, only set a mark
		if pr.isDeleted() {
			return
		}
		pr.setDeleted(true)
		// remove handler from map
		pr.Remove()
	}()
}

// mark deleted, not used this time
func (pr *handlerPrv) setDeleted(deleted bool) {
	pr.lock.Lock()
	defer pr.lock.Unlock()
	pr.deleted = deleted
}

// mark deleted
func (pr *handlerPrv) isDeleted() bool {
	pr.lock.Lock()
	defer pr.lock.Unlock()
	deleted := pr.deleted
	return deleted
}

// close handler
func (pr *handlerPrv) Close() {
	if pr.lConn != nil {
		_ = pr.lConn.Close()
	}
	if pr.rConn != nil {
		_ = pr.rConn.Close()
	}
	logger.Debugf("[%s] proxy has successfully closed, local [%s] -> remote [%s]", pr.typ, pr.lAddr.String(), pr.rAddr.String())
}

// close and delete handler from manager
func (pr *handlerPrv) Remove() {
	pr.mgr.CloseBaseHandler(pr.scope, pr.key)
}
