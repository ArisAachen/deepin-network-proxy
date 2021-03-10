package Config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/DeepinProxy/Com"
	define "github.com/DeepinProxy/Define"
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
}

// scope proxy
type ScopeProxies struct {
	Proxies map[string][]Proxy `yaml:"proxies"` // map[http,sock4,sock5][]proxy
	// proxy setting
	ProxyProgram   []string `yaml:"proxy-program"`    // global proxy will ignore
	NoProxyProgram []string `yaml:"no-proxy-program"` // app proxy will ignore

	// white list
	WhiteList []string `yaml:"whitelist"` // white site dont use proxy, not use this time
	TPort     int      `yaml:"t-port"`
}

func (p *ScopeProxies) GetProxy(proto string, name string) (Proxy, error) {
	if p == nil {
		return Proxy{}, errors.New("proxy proxies is nil")
	}
	// get proxies
	proxies, ok := p.Proxies[proto]
	if !ok {
		return Proxy{}, fmt.Errorf("proxy proto [%s] not exist in proxies", proto)
	}
	// search name
	for _, proxy := range proxies {
		if proxy.Name == name {
			return proxy, nil
		}
	}
	return Proxy{}, fmt.Errorf("proxy name [%s] not exist in proto [%s]", name, proto)
}

// set and add proxy
func (p *ScopeProxies) SetProxy(proto string, name string, proxy Proxy) {
	if p == nil {
		return
	}
	// get proxies
	proxies, ok := p.Proxies[proto]
	if !ok {
		proxies = make([]Proxy, 10)
		proxies = append(proxies, proxy)
		p.Proxies[proto] = proxies
		return
	}
	var exist bool
	// search name and replace
	for index, old := range proxies {
		if old.Name == name {
			proxies[index] = proxy
			exist = true
		}
	}
	// if not exist, append
	if !exist {
		proxies = append(proxies, proxy)
	}
	p.Proxies[proto] = proxies
}

// proxy config
type ProxyConfig struct {
	AllProxies map[string]ScopeProxies `yaml:"all-proxies"` // map[global,app]ScopeProxies
}

// create new
func NewProxyCfg() *ProxyConfig {
	cfg := &ProxyConfig{
		AllProxies: make(map[string]ScopeProxies),
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

// get proxy by scope
func (p *ProxyConfig) GetScopeProxies(scope define.Scope) (ScopeProxies, error) {
	// check if all proxies is nil
	if p.AllProxies == nil {
		return ScopeProxies{}, errors.New("all proxy config is nil")
	}
	proxies, ok := p.AllProxies[scope.ToString()]
	if !ok {
		return ScopeProxies{}, fmt.Errorf("proxy scope [%s] cant found any proxy in all proxies map", scope)
	}
	return proxies, nil
}

func (p *ProxyConfig) SetScopeProxies(scope define.Scope, proxies ScopeProxies) {
	// check if all proxies is nil
	if p.AllProxies == nil {
		return
	}
	// set proxies
	p.AllProxies[scope.ToString()] = proxies
}

// get proxy from config map, index: [global,app] -> [http,sock4,sock5] -> [proxy-name]
func (p *ProxyConfig) GetProxy(scope string, proto string, name string) (Proxy, error) {
	// get global or app proxies from all proxies
	scopeProxy, ok := p.AllProxies[scope]
	if !ok {
		return Proxy{}, fmt.Errorf("proxy type [%s] cant found any proxy in all proxies map", scope)
	}
	// get http sock4 sock5 proxies from type proxies
	proxy, err := scopeProxy.GetProxy(proto, name)
	if err != nil {
		return Proxy{}, fmt.Errorf("proxy protocol [%s] cant found any proxy in [%s] map", proto, scope)
	}
	// proxy found
	return proxy, nil
}
