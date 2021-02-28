package Iptables

import (
	"os/exec"
	"strconv"
	"strings"
)

type RuleCommand struct {
	// soft   iptables or ip6tables
	soft string

	// body
	table     string    // raw mangle filter...
	operation Operation // append insert delete...
	chain     string    // OUTPUT INPUT...
	index     int       // 0 1 2
	contain   containRule
}

// run command
func (r *RuleCommand) CombinedOutput() ([]byte, error) {
	// param
	param := strings.Join(r.StringSl(), " ")
	cmd := exec.Command("/bin/bash", "-c", r.soft, param)
	logger.Debugf("run command %s", cmd.String())
	return cmd.CombinedOutput()
}

//


// make string
func (r *RuleCommand) StringSl() []string {
	// head
	result := []string{"-t", r.table, "-" + r.operation.ToString(), r.chain}
	if r.index != 0 {
		result = append(result, strconv.Itoa(r.index))
	}
	// body
	result = append(result, r.contain.StringSl()...)
	return result
}
