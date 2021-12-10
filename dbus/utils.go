package DBus

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	config "github.com/ArisAachen/deepin-network-proxy/config"
	"github.com/linuxdeepin/go-lib/appinfo/desktopappinfo"
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
	// get absolute path
	if !filepath.IsAbs(table) {
		// if is not absolute
		table, err = exec.LookPath(table)
		if err != nil {
			logger.Warningf("look path failed, err: %v", err)
			return "", err
		}
	}
	stat, err := os.Lstat(table)
	if err != nil {
		logger.Warningf("exe path %s error, err: %v", table, err)
		return "", err
	}
	if stat.Mode()&os.ModeSymlink != 0 {
		logger.Debugf("exe path %s is link, get real path", table)
		table, err = os.Readlink(table)
		if err != nil {
			logger.Warningf("read link failed, err: %v", err)
			return "", err
		}
	}
	return table, nil
}
