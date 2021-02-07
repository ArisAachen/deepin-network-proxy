package Config

import (
	"log"
	"testing"
)

func TestProxyConfig_LoadPxyCfg(t *testing.T) {


	cfg := &ProxyConfig{
		AllProxies: map[string]map[string][]Proxy{
			"global": ProxyForm{
				ProxyType{
					Type: "http",
					TypeSl: Proxies{
						Proxy{
							Name:           "http_one",
							Server:         "http://172.16.82.129",
							Port:           80,
							UserName:       "uos",
							Password:       "12345678",
							ProxyProgram:   []string{"apt", "ssr"},
							NoProxyProgram: []string{"apt", "ssr"},
							WhiteList:      []string{"baidu.com", "si.com"},
						},
						Proxy{
							Name:           "http_two",
							Server:         "http://172.16.82.129",
							Port:           80,
							UserName:       "uos",
							Password:       "12345678",
							ProxyProgram:   []string{"apt", "ssr"},
							NoProxyProgram: []string{"apt", "ssr"},
							WhiteList:      []string{"baidu.com", "si.com"},
						},
					},
				},
				ProxyType{
					Type: "sock4",
					TypeSl: Proxies{
						Proxy{
							Name:           "http_three",
							Server:         "http://172.16.82.129",
							Port:           80,
							UserName:       "uos",
							Password:       "12345678",
							ProxyProgram:   []string{"apt", "ssr"},
							NoProxyProgram: []string{"apt", "ssr"},
							WhiteList:      []string{"baidu.com", "si.com"},
						},
						Proxy{
							Name:           "http_four",
							Server:         "http://172.16.82.129",
							Port:           80,
							UserName:       "uos",
							Password:       "12345678",
							ProxyProgram:   []string{"apt", "ssr"},
							NoProxyProgram: []string{"apt", "ssr"},
							WhiteList:      []string{"baidu.com", "si.com"},
						},
					},
				},
			},
		},
		TPort: 8080,
	}

	err := cfg.WritePxyCfg("~/Desktop/ProxyNew.yaml")
	if err != nil {
		log.Fatal(err)
	}
}
