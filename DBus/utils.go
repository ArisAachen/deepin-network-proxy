package DBus

import (
	"encoding/json"
	config "github.com/DeepinProxy/Config"
	"os"
	"os/exec"
	"path/filepath"
	"pkg.deepin.io/lib/appinfo/desktopappinfo"
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
	// make desktop app info message
	appInfo, err := desktopappinfo.NewDesktopAppInfoFromFile(app)
	if err != nil {
		logger.Warningf("read desktop file failed, err: %v", err)
		return "", err
	}
	// get table
	table := appInfo.GetExecutable()
	// check if is absolute path
	if filepath.IsAbs(table) {
		// check if exist
		_, err := os.Stat(table)
		if err != nil {
			logger.Warningf("exe path %s error, err: %v", table, err)
			return "", err
		}
		return table, nil
	}
	// if is not absolute
	table, err = exec.LookPath(table)
	if err != nil {
		logger.Warningf("look path failed, err: %v", err)
		return "", err
	}
	return table, err
}
