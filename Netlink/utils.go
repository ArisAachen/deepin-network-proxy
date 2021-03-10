package Netlink

import (
	"errors"
	com "github.com/DeepinProxy/Com"
	"io/ioutil"
	"os"
	"path/filepath"
)

// get proc message
func getProcMsg(pid string) (ProcMessage, error) {
	if !com.IsPid(pid) {
		return ProcMessage{}, errors.New("dir is not proc path")
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
		return ProcMessage{}, errors.New("exe path is nil")
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

func getCPUTime() {

}

//// use to attach pid to cgroup
//func AttachCGroup(pid string, path string) error {
//	args := []string{"echo", pid, ">", path}
//	cmd := exec.Command("/bin/sh", "-c", strings.Join(args, " "))
//	logger.Debugf("start to attach cgroup %s", cmd.String())
//	buf, err := cmd.CombinedOutput()
//	if err != nil {
//		logger.Warningf("exec add cgroup failed, err: %v", err)
//		return err
//	}
//	logger.Debugf("result is %s", string(buf))
//	return nil
//}
