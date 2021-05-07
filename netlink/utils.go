package Netlink

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	com "github.com/ArisAachen/deepin-network-proxy/com"
)

// get proc message
func getProcMsg(pid string) (ProcMessage, error) {
	if !com.IsPid(pid) {
		return ProcMessage{}, errors.New("proc pid not exist")
	}
	// get read proc path
	exePath := filepath.Join(ProcDir, pid, exe)
	readExecPath, _ := os.Readlink(exePath)
	// read cgroups message
	cgPath := filepath.Join(ProcDir, pid, cgroup)
	buf, _ := ioutil.ReadFile(cgPath)
	cgroupPath := com.ParseCGroup2FromBuf(buf)
	// read ppid message
	statusPath := filepath.Join(ProcDir, pid, status)
	buf, _ = ioutil.ReadFile(statusPath)
	ppid := com.ParsePPidFromBuf(buf)

	// sometimes /proc/Pid/exe dont is empty link
	if readExecPath == "" {
		logger.Debugf("[%s] dont contain exe path", exePath)
		return ProcMessage{}, errors.New("exe path dont exist")
	}
	logger.Debugf("Pid [%s], exe [%s]", pid, readExecPath)
	// proc message
	msg := ProcMessage{
		ExecPath:    readExecPath,
		Cgroup2Path: cgroupPath,
		Pid:         pid,
		PPid:        ppid,
	}
	return msg, nil
}