package Route

// run command to route
type RunCommand struct {
	soft string // ip rule and ip route

	action string
	mark   string
	table  string
}