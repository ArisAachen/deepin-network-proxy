package main

import (
	proxyDBus "github.com/DeepinProxy/DBus"
	"log"
)

func main() {

	//
	manager := proxyDBus.NewManager()
	// load config
	err := manager.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	// export dbus service
	err = manager.Export()
	if err != nil {
		log.Fatal(err)
	}
	// wait
	manager.Wait()
	//err := bus.CreateProxyService()
	//if err != nil {
	//	return
	//}

	//err := netlink.CreateProcsService()
	//if err != nil {
	//	return
	//}

	//file, err := os.Open("")
	//if err != nil {
	//	log.Println(err)
	//	return
	//}

	//reader := bufio.NewReader(file)
	//writer := bufio.NewWriter(file)
	//
	//rw := bufio.NewReadWriter(reader, writer)
	//
	//for {
	//	line, _, err := rw.ReadLine()
	//	if err != nil {
	//
	//	}
	//
	//	if string(line) == "" {
	//
	//	}
	//
	//}

	//time.Sleep(24 * time.Hour)
}
