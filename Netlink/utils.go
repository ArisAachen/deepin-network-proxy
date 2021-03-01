package Netlink

import (
	"errors"
	"io/ioutil"
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
	exePath := filepath.Join(ProcDir, pid, exe)
	readExecPath, _ := os.Readlink(exePath)
	cgPath := filepath.Join(ProcDir, pid, cgroup)
	buf, _ := ioutil.ReadFile(cgPath)
	cgroupPath := com.ParseCGroup2FromBuf(buf)

	// sometimes /proc/Pid/exe dont is empty link
	if readExecPath == "" {
		logger.Debugf("[%s] dont contain exe path", exePath)
		return ProcMessage{}, errors.New("exe path is nil")
	}
	logger.Debugf("Pid [%s], exe [%s]", pid, readExecPath)
	// proc message
	msg := ProcMessage{
		ExecPath:    readExecPath,
		Cgroup2Path: string(cgroupPath),
		Pid:         pid,
	}
	return msg, nil
}
