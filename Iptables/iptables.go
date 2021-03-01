package Iptables

import (
	"errors"
	"pkg.deepin.io/lib/log"
)

// cant extends
var defaultTables = []string{
	"raw",
	"mangle",
	"nat",
	"filter",
}

// can extends
var defaultChains = []string{
	"PREROUTING",
	"INPUT",
	"FORWARD",
	"OUTPUT",
	"POSTROUTING",
}

type TablesManager struct {
	tableSl []*TableRule
}

// tables manager to manager table rules
func NewTablesManager() *TablesManager {
	// table manager
	tableMgr := &TablesManager{
		tableSl: []*TableRule{},
	}
	// init default rules and chains
	tableMgr.InitRules()
	return tableMgr
}

// init default iptables rules
func (m *TablesManager) InitRules() {
	// init default tables
	for _, table := range defaultTables {
		// add table to table manager
		tbRule := NewTableRule(table)
		// init default chains to table
		for _, chain := range defaultChains {
			switch chain {
			// init PREROUTING chain    raw mangle nat
			case "PREROUTING":
				if table == "raw" || table == "mangle" || table == "nat" {
					logger.Debugf("[%s] table add chain [%s]", table, chain)
					cnRule := &ChainRule{
						chain: chain,
					}
					tbRule.chains[chain] = cnRule
				}
				break
				// init INPUT chain     mangle filter
			case "INPUT":
				if table == "filter" || table == "mangle" {
					logger.Debugf("[%s] table add chain [%s]", table, chain)
					cnRule := &ChainRule{
						chain: chain,
					}
					tbRule.chains[chain] = cnRule
				}
				break
				// init FORWARD chain     mangle filter
			case "FORWARD":
				if table == "filter" || table == "mangle" {
					logger.Debugf("[%s] table add chain [%s]", table, chain)
					cnRule := &ChainRule{
						chain: chain,
					}
					tbRule.chains[chain] = cnRule
				}
				break
				// init OUTPUT   all support
			case "OUTPUT":
				logger.Debugf("[%s] table add chain [%s]", table, chain)
				cnRule := &ChainRule{
					chain: chain,
				}
				tbRule.chains[chain] = cnRule
				break
				// init OUTPUT  mangle nat
			case "POSTROUTING":
				if table == "nat" || table == "mangle" {
					logger.Debugf("[%s] table add chain [%s]", table, chain)
					cnRule := &ChainRule{
						chain: chain,
					}
					tbRule.chains[chain] = cnRule
				}
				break
			default:
				logger.Warningf("init unknown chain [%s]", chain)
			}
		}
		if tbRule != nil {
			m.tableSl = append(m.tableSl, tbRule)
		}
	}
}

// create new chain,    new chain show attach to default
func (m *TablesManager) CreateChain(table string, parent string, index int, chain string, base []BaseRule, extends []ExtendsRule) error {
	// check if create default chain

	// search table
	var tbRule *TableRule
	for _, rule := range m.tableSl {
		if rule.table == table {
			tbRule = rule
			logger.Debugf("table [%s] found, allow to add", table)
			break
		}
	}
	// check if table name exist
	if tbRule == nil {
		return errors.New("table dont exist")
	}
	// add chain to table
	return tbRule.CreateChain(parent, index, chain, base, extends)
}

// add rule
func (m *TablesManager) AddRule(table string, chain string, action string, base []BaseRule, extends []ExtendsRule) error {
	var tbRule *TableRule
	for _, rule := range m.tableSl {
		if rule.table == table {
			logger.Debugf("table [%s] found, allow to add rule", table)
			tbRule = rule
			break
		}
	}
	// check if table name exist
	if tbRule == nil {
		return errors.New("table dont exist")
	}
	return tbRule.AddRule(chain, action, base, extends)
}

func init() {
	logger = log.NewLogger("daemon/iptables")
	logger.SetLogLevel(log.LevelDebug)
}
