package Iptables

import (
	"errors"
	com "github.com/DeepinProxy/Com"
	"pkg.deepin.io/lib/log"
)

/*
	Iptables module extends
	1. linux net flow redirect (now support)
	2. transparent proxy (now support)
	3. firewall (now support)
	4. ipv4 (now support)       // iptables    may use nf_tables
	5. ipv6 (not support yet)   // ip6tables   may use nf_tables
*/

var logger *log.Logger

// action
type Action int

const (
	ACCEPT Action = iota
	DROP
	RETURN
	QUEUE
	REDIRECT
	MARK
)

func (a Action) ToString() string {
	switch a {
	case ACCEPT:
		return "ACCEPT"
	case DROP:
		return "DROP"
	case RETURN:
		return "RETURN"
	case QUEUE:
		return "QUEUE"
	case REDIRECT:
		return "REDIRECT"
	case MARK:
		return "MARK"
	default:
		return ""
	}
}

// base rule
type baseRule struct {
	match string // -s
	param string // 1111.2222.3333.4444
}

// make string
func (bs *baseRule) String() string {
	return "-" + bs.match + " " + bs.param
}

// extends elem
type extendsElem struct {
	match string     // mark
	base  []baseRule // --mark 1
}

// make string   mark --mark 1
func (elem *extendsElem) StringSl() []string {
	// match
	result := []string{elem.match}
	// param
	for _, bs := range elem.base {
		result = append(result, "--"+bs.match, bs.param)
	}
	return result
}

// extends rule
type extendsRule struct {
	match string        // -m
	base  []extendsElem // mark --mark 1
}

// make string  -m mark --mark 1
func (ex *extendsRule) StringSl() []string {
	// match
	result := []string{"-" + ex.match}
	// param
	for _, elem := range ex.base {
		result = append(result, elem.StringSl()...)
	}
	return result
}

// container to contain one complete rule
type containRule struct {
	action  Action // ACCEPT DROP RETURN QUEUE REDIRECT MARK
	bsRules []baseRule
	exRules []extendsRule
}

// make string    -j ACCEPT -s 1111.2222.3333.4444 -m mark --set-mark 1
func (cn *containRule) StringSl() []string {
	var result []string
	result = append(result, "-j", cn.action.ToString())
	// add base rules
	for _, bs := range cn.bsRules {
		result = append(result, bs.String())
	}
	// add extends rules
	for _, ex := range cn.exRules {
		result = append(result, ex.StringSl()...)
	}
	return result
}

// table rule contains many chains
type TableRule struct {
	table  string // table name:  raw mangle filter nat
	chains map[string]ChainRule
}

// chain rule
type ChainRule struct {
	chain     string // chain name: PREROUTING INPUT FORWARD OUTPUT POSTROUTING
	parent    string // parent chain name, if has not, is ""
	containSl []containRule
}

// set parent if need
func (c *ChainRule) SetParent(parent string) {
	c.parent = parent
}

// exec iptables command and add to record
func (c *ChainRule) AddRule(action Action, base []baseRule, extends []extendsRule) error {
	// make contain
	contain := containRule{
		action:  action,
		bsRules: base,
		exRules: extends,
	}
	// mega add slice
	ifc, update, err := com.MegaAdd(c.containSl, contain)
	if err != nil {
		logger.Warningf("[%s] add rule failed, err: %v", c.chain, err)
	}
	// check if already exist
	if !update {
		logger.Debugf("[%s] dont need add rule, already exist", c.chain)
		return nil
	}
	// check type
	temp, ok := ifc.([]containRule)
	if !ok {
		logger.Warningf("[%s] add rule failed, convert type failed", c.chain)
		return errors.New("convert type failed")
	}
	c.containSl = temp
	logger.Debugf("[%s] add rule success, rule: %s ", c.chain, contain.StringSl())
	return nil
}

// create table rule
func NewTableRule(name string) *TableRule {
	rule := &TableRule{
		table: name,
	}
	return rule
}

func (t *TableRule) AddRule(chain string, action Action, base []baseRule, extends []extendsRule) {


}

func init() {
	logger = log.NewLogger("daemon/proxy")
	logger.SetLogLevel(log.LevelDebug)
}
