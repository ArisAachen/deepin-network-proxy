package UdpProxy

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/DeepinProxy/Config"
	"pkg.deepin.io/lib/log"
)

type Sock5Handler struct {
	localHandler  net.Addr
	RemoteHandler net.Conn
	proxy         Config.Proxy
	data          sync.Pool
}

var logger *log.Logger

func NewSock5Handler(local net.Addr, proxy Config.Proxy) *Sock5Handler {
	handler := &Sock5Handler{
		localHandler: local,
		proxy:        proxy,
		data: sync.Pool{
			New: func() interface{} {
				b := make([]byte, 512)
				return &b
			},
		},
	}
	return handler
}

// auth message
type auth struct {
	user     string
	password string
}

// create tunnel between proxy and server
func (handler *Sock5Handler) Tunnel(rConn net.Conn, addr *net.UDPAddr) error {
	// auth message
	auth := auth{
		user:     handler.proxy.UserName,
		password: handler.proxy.Password,
	}
	/*
	    sock5 client hand shake request
	  +----+----------+----------+
	  |VER | NMETHODS | METHODS  |
	  +----+----------+----------+
	  | 1  |    1     | 1 to 255 |
	  +----+----------+----------+
	*/
	// sock5 proto
	// buffer := new(bytes.Buffer)
	buf := make([]byte, 3)
	buf[0] = 5
	buf[1] = 1
	buf[2] = 0
	if auth.user != "" && auth.password != "" {
		buf[1] = 2
		buf = append(buf, byte(2))
	}
	// sock5 hand shake
	_, err := rConn.Write(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 hand shake request failed, err: %v", err)
		return err
	}
	/*
		sock5 server hand shake response
		+----+--------+
		|VER | METHOD |
		+----+--------+
		| 1  |   1    |
		+----+--------+
	*/
	_, err = rConn.Read(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 hand shake response failed, err: %v", err)
		return err
	}
	logger.Debugf("[udp] sock5 hand shake response success message auth method: %v", buf[1])
	if buf[0] != 5 || (buf[1] != 0 && buf[1] != 2) {
		return fmt.Errorf("sock5 proto is invalid, sock type: %v, method: %v", buf[0], buf[1])
	}
	// check if server need auth
	if buf[1] == 2 {
		/*
		    sock5 auth request
		  +----+------+----------+------+----------+
		  |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
		  +----+------+----------+------+----------+
		  | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
		  +----+------+----------+------+----------+
		*/
		buf = make([]byte, 1)
		buf[0] = 1
		buf = append(buf, byte(len(auth.user)))
		buf = append(buf, []byte(auth.user)...)
		buf = append(buf, byte(len(auth.password)))
		buf = append(buf, []byte(auth.password)...)
		// write auth message to writer
		_, err = rConn.Write(buf)
		if err != nil {
			logger.Warningf("[udp] sock5 auth request failed, err: %v", err)
			return err
		}
		buf = make([]byte, 32)
		_, err = rConn.Read(buf)
		if err != nil {
			logger.Warningf("[udp] sock5 auth response failed, err: %v", err)
			return err
		}
		// RFC1929 user/pass auth should return 1, but some sock5 return 5
		if buf[0] != 5 && buf[0] != 1 {
			logger.Warningf("[udp] sock5 auth response incorrect code, code: %v", buf[0])
			return fmt.Errorf("incorrect sock5 auth response, code: %v", buf[0])
		}
		logger.Debugf("[udp] sock5 auth success, code: %v", buf[0])
	}
	/*
			sock5 connect request
		   +----+-----+-------+------+----------+----------+
		   |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
		   +----+-----+-------+------+----------+----------+
		   | 1  |  1  | X'00' |  1   | Variable |    2     |
		   +----+-----+-------+------+----------+----------+
	*/
	// start create tunnel
	buf = make([]byte, 4)
	buf[0] = 5
	buf[1] = 3 // udp
	buf[2] = 0 // reserved
	// add addr
	if addr.IP.To4() != nil {
		buf[3] = 1
		buf = append(buf, net.IP{127, 0, 0, 0}...)
	} else if addr.IP.To16() != nil {
		buf[3] = 4
		buf = append(buf, addr.IP.To16()...)
	} else {
		buf[3] = 3
		buf = append(buf, addr.IP...)
	}
	// convert port 2 byte
	portU := uint16(addr.Port)
	if portU == 0 {
		portU = 80
	}
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, portU)
	buf = append(buf, port...)
	// request proxy connect remote server
	logger.Debugf("[udp] sock5 send connect request, buf: %v", buf)
	_, err = rConn.Write(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 send connect request failed, err: %v", err)
		return err
	}
	logger.Debugf("[udp] sock5 request successfully")
	buf = make([]byte, 16)
	_, err = rConn.Read(buf)
	if err != nil {
		logger.Warningf("[udp] sock5 connect response failed, err: %v", err)
		return err
	}
	logger.Debugf("[udp] sock5 response successfully, buf: %v", buf)
	if buf[0] != 5 || buf[1] != 0 {
		logger.Warningf("[udp] sock5 connect response failed, version: %v, code: %v", buf[0], buf[1])
		return fmt.Errorf("[udp] incorrect sock5 connect reponse, version: %v, code: %v", buf[0], buf[1])
	}
	// dial remote udp server
	udpServer := net.UDPAddr{
		IP:   buf[4:8],
		Port: int(binary.BigEndian.Uint16(buf[8:10])),
	}
	udpConn, err := net.Dial("udp", udpServer.String())
	if err != nil {
		logger.Warningf("[udp] dial remote udp failed, err: %v", err)
		return err
	}

	logger.Debugf("[udp] sock5 proxy: tunnel create success, [%s] -> [%s] -> [%s]",
		handler.localHandler.String(), udpServer.String(), addr.String())
	// save remote handler

	handler.RemoteHandler = udpConn
	return nil
}

/*
	DataGram
	| Head | Data |
	| Size (uint16) | Data |
*/

func (handler *Sock5Handler) PutData(buf []byte) {
	handler.data.Put(buf)
}

func (handler *Sock5Handler) Get() []byte {
	iBuf, ok := handler.data.Get().([]byte)
	if !ok {
		logger.Warningf("get sock data is not []byte")
	}
	return iBuf
}
