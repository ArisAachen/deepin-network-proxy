package Config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// proxy type, support HTTP SOCK4 SOCK5
type ProxyProto int

const (
	HttpProxy ProxyProto = iota
	Sock4Proxy
	Sock5Proxy
)

func (p ProxyProto) String() string {
	var proto string
	switch p {
	case HttpProxy:
		proto = "HTTP"
	case Sock4Proxy:
		proto = "SOCK4"
	case Sock5Proxy:
		proto = "SOCK5"
	default:
		proto = "UNKNOWN"
	}
	return proto
}

// proxy proto, support tcp udp
type ProxyNetwork int

const (
	TcpProxy ProxyNetwork = iota
	UdpProxy
)

func (p ProxyNetwork) String() string {
	var network string
	switch p {
	case TcpProxy:
		network = "tcp"
	case UdpProxy:
		network = "udp"
	default:
		network = "unknown"
	}
	return network
}

/*
	one example for config

# proxy form global or app
global:
  type: "http"
  proxies:
   - name: "http_one"
     server: "http://172.16.82.129"
     port: 80
     username: "uos"
     password: "12345678"
     proxy_program: ""  # useless for global
     no_proxy_program: "/opt/apps/com.163.music/files/bin/netease-cloud-music"
     whitelist: "https://baidu.com"

app:
  type: "sock5"
  proxies:
   - name: "http_one"
     server: "http://172.16.82.129"
     port: 80
     username: "uos"
     password: "12345678"
     proxy_program: ""  # useless for global
     no_proxy_program: "/opt/apps/com.163.music/files/bin/netease-cloud-music"
     whitelist: "https://baidu.com"

*/

// proxy type
type Proxy struct {
	// ProtoType      string `json:"type"` // http sock4 sock5
	Name           string   `yaml:"name"`
	Server         string   `yaml:"server"`
	Port           int      `yaml:"port"`
	UserName       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	ProxyProgram   []string `yaml:"proxy-program"`
	NoProxyProgram []string `yaml:"no-proxy-program"` // app dont use proxy
	WhiteList      []string `yaml:"whitelist"`        // white site dont use proxy
}

// single proxy type slice
type Proxies []Proxy

// proxy type    http sock4 sock5
type ProxyType struct {
	Type   string  `yaml:"type"`
	TypeSl Proxies `yaml:"proxies"`
}

type ProxyForm []ProxyType

// proxy config
type ProxyConfig struct {
	Member map[string]ProxyForm `yaml:"proxy-form"` // map[global,app]map[http,sock4,sock5][]proxy
}

// create new
func NewProxyCfg() *ProxyConfig {
	cfg := new(ProxyConfig)
	return cfg
}

// write config file
func (p *ProxyConfig) WritePxyCfg(path string) error {
	// marshal interface
	buf, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	// in case delete by other user
	err = ioutil.WriteFile(path, buf, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// load config file
func (p *ProxyConfig) LoadPxyCfg(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("absoute file path failed, err: %v", err)
	}
	// read file
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return errors.New("config file not exist")
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file failed, err: %v", err)
	}
	// unmarshal config file
	err = yaml.Unmarshal(buf, p)
	if err != nil {
		return fmt.Errorf("unmarshal config file failed, err: %v", err)
	}
	// (host)
	return nil
}

func (p *ProxyConfig) GetProxy(form string, proto string) (Proxy, error) {
	// get proxy form
	member, ok := p.Member[form]
	if !ok {
		return Proxy{}, errors.New("form dont found in proxy map")
	}
	// find proto
	var typeSl Proxies
	for _, py := range member {
		if py.Type == proto {
			typeSl = py.TypeSl
		}
	}
	// check if exist
	if typeSl == nil {
		return Proxy{}, errors.New("proto dont found in proxy member")
	}
	// find rule
	return typeSl[0], nil
}