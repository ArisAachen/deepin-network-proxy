package Netlink

import (
	"errors"
	"os"
	"path/filepath"

	com "github.com/DeepinProxy/Com"
)

// get proc message
func getProcMsg(pid string) (ProcMessage, error) {
	if !com.IsPid(pid) {
		return ProcMessage{}, errors.New("dir is not proc path")
	}
	// get read proc path
	exePath := filepath.Join(ProcDir , pid , exe)
	readExecPath, _ := os.Readlink(exePath)
	cwdPath := filepath.Join(ProcDir , pid , cwd)
	cwdRealPath, _ := os.Readlink(cwdPath)
	// sometimes /proc/pid/exe dont is empty link
	if readExecPath == "" {
		logger.Debugf("[%s] dont contain exe path", exePath)
		return ProcMessage{}, errors.New("exe path is nil")
	}
	logger.Debugf("pid [%s], exe [%s]", pid, readExecPath)
	// proc message
	msg := ProcMessage{
		execPath: readExecPath,
		cwdPath:  cwdRealPath,
		pid:      pid,
	}
	return msg, nil
}