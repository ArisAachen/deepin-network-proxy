package Config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/DeepinProxy/Com"
	"gopkg.in/yaml.v2"
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
	// proxy proto type
	ProtoType string `json:"type"` // http sock4 sock5

	// [proto]&[name] as ident
	Name string `yaml:"name"`

	// proxy server
	Server string `yaml:"server"`
	Port   int    `yaml:"port"`

	// auth message
	UserName string `yaml:"username"`
	Password string `yaml:"password"`

	// proxy setting
	ProxyProgram   []string `yaml:"proxy-program"`    // global proxy will ignore
	NoProxyProgram []string `yaml:"no-proxy-program"` // app proxy will ignore

	// white list
	WhiteList []string `yaml:"whitelist"` // white site dont use proxy, not use this time
}

// proxy config
type ProxyConfig struct {
	AllProxies map[string]map[string][]Proxy `yaml:"all-proxy"` // map[global,app]map[http,sock4,sock5][]proxy
	TPort      int                           `yaml:"t-port"`
}

// create new
func NewProxyCfg() *ProxyConfig {
	cfg := &ProxyConfig{
		AllProxies: make(map[string]map[string][]Proxy),
	}
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
	// guarantee file dir is exist
	err = Com.GuaranteeDir(filepath.Dir(path))
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

// get proxy from config map, index: [global,app] -> [http,sock4,sock5] -> [proxy-name]
func (p *ProxyConfig) GetProxy(typ string, proto string, name string) (Proxy, error) {
	var proxy Proxy
	// get global or app proxies from all proxies
	typProxies, ok := p.AllProxies[typ]
	if !ok {
		return proxy, fmt.Errorf("proxy type [%s] cant found any proxy in all proxies map", typ)
	}
	// get http sock4 sock5 proxies from type proxies
	proxies, ok := typProxies[proto]
	if !ok {
		return proxy, fmt.Errorf("proxy protocol [%s] cant found any proxy in [%s] map", proto, typ)
	}
	// get proxy from proxy slice
	for _, elem := range proxies {
		if elem.Name == name {
			proxy = elem
		}
	}
	// check if find proxy, if equal with empty one, means proxy cant found in map
	if reflect.DeepEqual(proxy, Proxy{}) {
		return proxy, fmt.Errorf("proxy name [%s] cant found any proxy in slice [%s] map[%s]", name, proto, typ)
	}
	// proxy found
	return proxy, nil
}
