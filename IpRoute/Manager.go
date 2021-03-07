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
func (m *Manager) CreateRoute(node RouteNodeSpec, info RouteInfoSpec) (*Route, error) {


	return nil, nil
}

func init() {
	logger = log.NewLogger("damon/route")
	logger.SetLogLevel(log.LevelDebug)
}
