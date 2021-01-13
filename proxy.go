package main

import (
	"golang.org/x/sys/unix"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var Head string = `
CONNECT 10.20.31.56:808 HTTP/1.1
Host: 10.20.31.56:808
`

func handleReadRequest(CConn net.Conn, fd int) {
	log.Println("remote addr is ", CConn.RemoteAddr())
	defer CConn.Close()
	for {
		buf := make([]byte, 512)
		_, err := CConn.Read(buf)
		if err != nil {
			log.Println("CConn read error ", err)
			return
		}
		log.Println("CConn read success ")
		log.Println(string(buf))
		log.Println("------------------Read end line------------------")

		_, err = unix.Write(fd, buf)
		if err != nil {
			log.Println("LConn read error ", err)
			return
		}
	}
}

func handleWriteRequest(CConn net.Conn, fd int) {
	defer CConn.Close()
	for {
		buf := make([]byte, 512)
		_, err := unix.Read(fd, buf)
		if err != nil {
			log.Println("LConn read error ", err)
			return
		}
		log.Println("LConn read success ")
		log.Println(string(buf))
		log.Println("------------------Write end line------------------")

		_, err = CConn.Write(buf)
		if err != nil {
			log.Println("LConn read error ", err)
			return
		}
	}

}

func main() {

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()

	for {
		log.Println("begin listening... ...")
		CConn, err := l.Accept()
		if err != nil {
			log.Println("accept is ", err)
			return
		}

		ipSli := strings.Split(CConn.RemoteAddr().String(), ":")
		if len(ipSli) < 0 {
			log.Fatal("split remote addr failed, \n", CConn.RemoteAddr().String())
		}
		log.Println("split remote addr success, remote addr: ", CConn.RemoteAddr().String(), " local addr: ", CConn.LocalAddr().String())

		ip := net.ParseIP(ipSli[0])
		if ip == nil {
			log.Fatal("parse ip failed, addr is \n", CConn.RemoteAddr().String())
		}

		var sock int
		var sockAddr unix.Sockaddr
		// ip is ipv4
		if ip.To4() != nil {
			sock, err = unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
			sockIpv4 := new(unix.SockaddrInet4)
			for i := 0; i < len(sockIpv4.Addr); i++ {
				sockIpv4.Addr[i] = ip[i]
			}
			sockIpv4.Port, err = strconv.Atoi(ipSli[1])
			if err != nil {
				log.Fatal("convert port failed, remote addr is: ", CConn.RemoteAddr().String(), " err: \n", err)
			}
			sockAddr = sockIpv4
		} else {
			// ip is ipv6
			sock, err = unix.Socket(unix.AF_INET6, unix.SOCK_STREAM, unix.IPPROTO_TCP)
			log.Println(">>>>>> ignore ipv6")
		}
		if err != nil {
			log.Fatal("create sock err: ", err)
		}

		err = unix.Connect(sock, sockAddr)
		if err != nil {
			log.Println("connect sock err: \n", err)
			continue
		}

		_, err = unix.Write(sock, []byte(Head))
		if err != nil {
			log.Fatal("write head err: \n", err)
		}

		if err != nil {
			log.Println("head write err is ", err)
			return
		}

		go handleReadRequest(CConn, sock)
		go handleWriteRequest(CConn, sock)
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
