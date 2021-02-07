package Config

import (
	"log"
	"testing"

	"github.com/DeepinProxy/Com"
)

func TestProxyConfig_LoadPxyCfg(t *testing.T) {
	http := []Proxy{
		// http proxy
		{
			ProtoType:      "http",
			Name:           "http_1",
			Server:         "10.20.31.132",
			Port:           808,
			UserName:       "uos",
			Password:       "12345678",
			ProxyProgram:   []string{"apt", "ssr"},
			NoProxyProgram: []string{"apt", "ssr"},
			WhiteList:      []string{"baidu.com", "si.com"},
		},
		{
			ProtoType:      "http",
			Name:           "http_2",
			Server:         "10.20.31.132",
			Port:           80,
			UserName:       "uos",
			Password:       "12345678",
			ProxyProgram:   []string{"apt", "ssr"},
			NoProxyProgram: []string{"apt", "ssr"},
			WhiteList:      []string{"baidu.com", "si.com"},
		},
	}
	sock4 := []Proxy{
		// http proxy
		{
			ProtoType:      "sock4",
			Name:           "sock4_1",
			Server:         "10.20.31.132",
			Port:           1080,
			UserName:       "uos",
			Password:       "12345678",
			ProxyProgram:   []string{"apt", "ssr"},
			NoProxyProgram: []string{"apt", "ssr"},
			WhiteList:      []string{"baidu.com", "si.com"},
		},
		{
			ProtoType:      "sock4",
			Name:           "sock4_2",
			Server:         "10.20.31.132",
			Port:           1080,
			UserName:       "uos",
			Password:       "12345678",
			ProxyProgram:   []string{"apt", "ssr"},
			NoProxyProgram: []string{"apt", "ssr"},
			WhiteList:      []string{"baidu.com", "si.com"},
		},
	}
	sock5 := []Proxy{
		// http proxy
		{
			ProtoType:      "sock5",
			Name:           "sock5_1",
			Server:         "10.20.31.132",
			Port:           1080,
			UserName:       "uos",
			Password:       "12345678",
			ProxyProgram:   []string{"apt", "ssr"},
			NoProxyProgram: []string{"apt", "ssr"},
			WhiteList:      []string{"baidu.com", "si.com"},
		},
		{
			ProtoType:      "sock5",
			Name:           "sock5_2",
			Server:         "10.20.31.132",
			Port:           1080,
			UserName:       "uos",
			Password:       "12345678",
			ProxyProgram:   []string{"apt", "ssr"},
			NoProxyProgram: []string{"apt", "ssr"},
			WhiteList:      []string{"baidu.com", "si.com"},
		},
	}

	cfg := &ProxyConfig{
		AllProxies: map[string]map[string][]Proxy{
			"global": {
				"http":  http,
				"sock4": sock4,
				"sock5": sock5,
			},
			"app": {
				"http":  http,
				"sock4": sock4,
				"sock5": sock5,
			},
		},
		TPort: 8080,
	}

	path, err := Com.GetUserConfigDir()
	if err != nil {
		return
	}
	err = cfg.WritePxyCfg(path)
	if err != nil {
		log.Fatal(err)
	}
}
