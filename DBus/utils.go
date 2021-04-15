package DBus

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	config "github.com/DeepinProxy/Config"
	"io"
	"io/ioutil"
	"strings"
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


// parse .desktop file to get real path
func parseDesktopPath(app string) (string, error) {
	if !strings.HasSuffix(app, ".desktop") {
		return app, nil
	}
	buf, err := ioutil.ReadFile(app)
	if err != nil {
		// if read failed, r
		logger.Warningf("read desktop file %s failed, err: %v", app, err)
		return "", err
	}
	reader := bufio.NewReader(bytes.NewBuffer(buf))
	for {
		msgBuf, _, err := reader.ReadLine()
		if err != io.EOF {
			logger.Debugf("read end file %s, cant find Exec path", app)
			return "", nil
		} else if err != nil {
			logger.Warningf("read buf %s failed, err: %v", app, err)
		}
		msg := string(msgBuf)
		// find Exec=/xxx/xxx xxx
		if strings.HasPrefix(msg, "Exec") {
			msgSl := strings.Split(msg, "=")
			if len(msgSl) < 2 {
				logger.Warningf("read path %s failed, dont include exec path", app)
				return "", errors.New("exec path dont defined")
			}
			exePathSl := strings.Split(msgSl[1], " ")
			return exePathSl[0], nil
		}
	}
}