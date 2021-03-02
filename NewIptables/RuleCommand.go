package NewIptables

type ruleCommand struct {
	// soft   iptables or ip6tables
	soft string

	// body
	table     string    // raw mangle filter...
	operation Operation // append insert delete...
	chain     string    // OUTPUT INPUT...
	index     int       // 0 1 2
	cpl       CompleteRule
}
