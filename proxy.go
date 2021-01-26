package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	cfg "github.com/proxyTest/Config"
	pro "github.com/proxyTest/HttpProxy"
)

func handleReadRequest(CConn net.Conn, LConn net.Conn) {
	for {
		buf := make([]byte, 512)
		_, err := CConn.Read(buf)
		if err != nil {
			log.Println("CConn read error ", err)
			return
		}

		_, err = LConn.Write(buf)
		if err != nil {
			log.Println("LConn read error ", err)
			return
		}
	}
}

func handleWriteRequest(CConn net.Conn, LConn net.Conn) {
	for {
		buf := make([]byte, 512)
		_, err := LConn.Read(buf)
		if err != nil {
			log.Println("LConn read error ", err)
			return
		}

		if strings.Index(string(buf), "Proxy-agent") != 0 {
			return
		}

		_, err = CConn.Write(buf)
		if err != nil {
			log.Println("LConn read error ", err)
			return
		}
	}

}

const SO_ORIGINAL_DST = 80

func main() {

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()

	//text := "aris:12345678"
	//encodedText := base64.StdEncoding.EncodeToString([]byte(text))

	config := cfg.NewProxyCfg()
	err = config.LoadPxyCfg("/home/aris/Desktop/Proxy.yaml")
	if err != nil {
		log.Fatal(err)
	}

	proxy, err := config.GetProxy("global", "http")
	if err != nil {
		log.Fatal(err)
	}

	for {
		CConn, err := l.Accept()
		if err != nil {
			log.Println("accept is ", err)
			return
		}

		handler := pro.NewTcpHandler(CConn, proxy)
		if handler == nil {
			continue
		}
		handler.Communicate()
		//tcpSock := CConn.(*net.TCPConn)
		//if tcpSock == nil {
		//	log.Fatal("convert to tcp sock failed")
		//}
		//
		//handler, err := tcpSock.File()
		//if err != nil {
		//	log.Fatal("get tcp fd failed")
		//}
		//
		//addr, err := unix.GetsockoptIPv6Mreq(int(handler.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//
		//LConn, err := net.Dial("tcp", "172.16.82.129:808")
		//if err != nil {
		//	log.Fatal(err)
		//}
		//
		//targetIp := net.IP(addr.Multiaddr[4:8])
		//targetPort := int(addr.Multiaddr[2])<<8 + int(addr.Multiaddr[3])
		//targetAddr := fmt.Sprintf("%s:%v", targetIp.String(), targetPort)
		//
		//req := &http.Request{
		//	Method: http.MethodConnect,
		//	Host:   targetAddr,
		//	URL: &url.URL{
		//		Host: targetAddr,
		//	},
		//	Header: http.Header{
		//		"Proxy-Connection":    []string{"Keep-Alive"},
		//		"Proxy-Authorization": []string{"Basic " + arr},
		//	},
		//}
		//
		//err = req.Write(LConn)
		//if err != nil {
		//	log.Fatal("write head failed, err ", err)
		//}
		//
		//buf := make([]byte, 512)
		//_, err = LConn.Read(buf)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//log.Println(string(buf))
		//
		//go io.Copy(LConn, CConn)
		//go io.Copy(CConn, LConn)
		//go handleReadRequest(CConn, LConn)
		//go handleWriteRequest(CConn, LConn)
	}

	//test := []byte{
	//	41,53,54,52,30,41,30,32,30,30}
	//log.Print(string(test))
	//
	//return
	//watcher, err := fsnotify.NewWatcher()
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	//err = watcher.Add("/var/lib/dpkg/lock-frontend")
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	//
	//go func() {
	//	time.Sleep(2 * time.Second)
	//	lockTest()
	//}()
	//
	//for {
	//	select {
	//	case event := <-watcher.Events:
	//		log.Println(event)
	//	case err = <-watcher.Errors:
	//		log.Println(err)
	//	}
	//}

	//// testNil()
	//client := &http.Client{
	//	CheckRedirect: func(req *http.Request, via []*http.Request) error {
	//		return http.ErrUseLastResponse
	//	},
	//}
	//res, err := client.Get(`http://detectportal.deepin.com`)
	//if err != nil {
	//	log.Printf("get failed ,err: %v \n", err)
	//	return
	//}
	//log.Println(res)
	////parse, err := url.Parse(`dawdawdaw.com`)
	////if err != nil {
	////	log.Println(err)
	////	return
	////}
	////log.Println(parse)
	//portal, err := getRedirectFromResponse(res)
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	//// portal = `baidu.com`
	//// 发起连接
	//err = exec.Command(`xdg-open`, portal).Start()
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	//log.Println(portal)
}

func lockTest() {
	f, err := os.OpenFile(`/var/lib/dpkg/lock-frontend`, os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Println(err)
		return
	}

	lock := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: io.SeekStart,
		Pid:    0,
		Len:    1,
		Start:  0,
	}
	err = syscall.FcntlFlock(f.Fd(), syscall.F_SETLKW, &lock)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("success lock")
	time.Sleep(10 * time.Second)
	args := []string{"-c", "apt install dealer"}
	out, err := exec.Command("/bin/bash", args...).CombinedOutput()
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("success install, %v \n", string(out))
}

//type TestStruct struct {
//	a string
//}
//
//func testNil() {
//	var test *TestStruct
//	log.Print(test.a)
//}

//func getRedirectFromResponse(resp *http.Response) (string, error) {
//	if resp == nil {
//		return "", errors.New("response is nil")
//	}
//	if resp.StatusCode != 301 && resp.StatusCode != 302 {
//		return "", errors.New("response is not redirect")
//	}
//	location := resp.Header.Get("Location")
//	if location == "" {
//		return "", errors.New("response has no location")
//	}
//	urls := strings.Split(location, "?")
//	return urls[0], nil
//}
