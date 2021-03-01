package CGroups

import (
	"os/exec"
)

// use to attach pid to cgroup
func AttachCGroup(path string, pid string) error {
	cmd := exec.Command("/bin/sh", "-c", "echo"+" "+pid+">"+path)
	logger.Debugf("start to attach cgroup %s", cmd.String())
	buf, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warningf("exec add cgroup failed, err: %v", err)
		return err
	}
	logger.Debugf("result is %s", string(buf))
	return nil
}

func ClassifyCGroup(cgroup string, path string) error {
	return nil
}
