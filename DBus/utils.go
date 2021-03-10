package DBus

import (
	"encoding/json"
	config "github.com/DeepinProxy/Config"
)

// unmarshal buf to proxy
func UnMarshalProxy(buf []byte) (config.Proxy, error) {
	proxy := config.Proxy{}
	err := json.Unmarshal(buf, &proxy)
	if err != nil {
		return config.Proxy{}, err
	}
	return proxy, nil
}
