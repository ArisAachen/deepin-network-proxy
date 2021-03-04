package Define

// proxy name
/*
	usage:
	1. use to create DBus project, to mark current proxy type
	2. use to create Iptables chain name
	3. use to create cgroups controller name
*/
const (
	Main   = "main"
	App    = "app"
	Global = "global"
)

// proxy type
/*
	usage:
	1. use to check recv dbus method
	2. use to make
	use to mark current proxy type
*/
const (
	// basic type
	HTTP  = "http"
	SOCK4 = "sock4"
	SOCK5 = "sock5"

	// extends type
	SOCK5UDP = "sock5-udp"
	SOCK5TCP = "sock5-tcp"
)

// proxy priority
const (
	MainPriority = iota
	AppPriority
	GlobalPriority
)
