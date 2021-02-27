package CGroups

import (
	"os/exec"
)

func AddCGroup(path string, pid string) error {


	cmd := exec.Command("echo", pid, ">", path)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warningf("exec add cgroup failed, err: %v", err)
		return err
	}
	logger.Debugf("result is %s", string(buf))
	return nil
}
