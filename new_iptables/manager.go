package NewIptables

import "github.com/linuxdeepin/go-lib/log"

/*
	Iptables module extends
	1. linux net flow redirect (now support)
	2. transparent proxy (now support)
	3. firewall (now support)
	4. ipv4 (now support)       // iptables    may use nf_tables
	5. ipv6 (not support yet)   // ip6tables   may use nf_tables
*/

// https://linux.die.net/man/8/iptables

var logger *log.Logger

var tableSl = map[string][]string{
	"raw": []string{
		"PREROUTING",
		"OUTPUT",
	},
	"mangle": []string{
		"PREROUTING",
		"INPUT",
		"FORWARD",
		"OUTPUT",
		"POSTROUTING",
	},
	"nat": []string{
		"PREROUTING",
		"OUTPUT",
		"POSTROUTING",
	},
	"filter": []string{
		"INPUT",
		"FORWARD",
		"OUTPUT",
	},
}

type Manager struct {
	tables map[string]*Table
}

// create manager
func NewManager() *Manager {
	manager := &Manager{
		tables: make(map[string]*Table),
	}
	return manager
}

// init table
func (m *Manager) Init() {
	logger.Debug("init manager")
	// init default table and chain
	for tName, cNameSl := range tableSl {
		// create tables to manager
		table := &Table{
			Name:   tName,
			chains: make(map[string]*Chain),
		}
		// create chain to table
		for _, cName := range cNameSl {
			// default chain dont need to create
			chain := &Chain{
				Name:      cName,
				table:     table,
				children:  make(map[string]*Chain),
				cplRuleSl: []*CompleteRule{},
			}
			// add chain to table
			table.chains[cName] = chain
			logger.Debugf("[%s] add default chain %s", tName, cName)
		}
		// add table to manager
		m.tables[tName] = table
	}
	return
}

// get chain, usually use to get default chain
func (m *Manager) GetChain(tName string, cName string) *Chain {
	// get table
	table, ok := m.tables[tName]
	if !ok {
		logger.Warningf("[%s] get table %s not exist", "manager", tName)
		return nil
	}
	// get chain
	chain, ok := table.chains[cName]
	if !ok {
		logger.Warningf("[%s] get table %s dont have chain %s", "manager", tName, cName)
		return nil
	}
	logger.Debugf("[%s] get table %s  get chain %s success", "manager", tName, cName)
	return chain
}

// init
func init() {
	logger = log.NewLogger("daemon/iptables")
	logger.SetLogLevel(log.LevelDebug)
}
