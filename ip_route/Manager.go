package IpRoute

import "pkg.deepin.io/lib/log"

var logger *log.Logger

type Manager struct {
	routes map[string]*Route
}

// create manager
func NewManager() *Manager {
	manager := &Manager{
		routes: make(map[string]*Route),
	}
	return manager
}

// create route
func (m *Manager) CreateRoute(name string, node RouteNodeSpec, info RouteInfoSpec) (*Route, error) {
	// create route
	route := &Route{
		table: name,
		Node:  node,
		Info:  info,
	}
	err := route.create()
	if err != nil {
		return nil, err
	}
	return route, nil
}

func init() {
	logger = log.NewLogger("damon/route")
	logger.SetLogLevel(log.LevelInfo)
}
