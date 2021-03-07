package IpRoute

import (
	"os/exec"
	"strings"
)

type action int

const (
	add action = iota
	del
)

func (a action) ToString() string {
	switch a {
	case add:
		return "add"
	case del:
		return "del"
	default:
		return ""
	}
}

// route node spec
type RouteNodeSpec struct {
	// args
	Type   string // unist local broadcast
	Prefix string // default 0/0 or ::/0
	Proto  string //
	Scope  string // global link host
	Metric string // metric num
}

// to string
func (node *RouteNodeSpec) String() string {
	args := []string{node.Type, node.Prefix}
	if node.Proto != "" {
		args = append(args, "protocol", node.Proto)
	}
	if node.Scope != "" {
		args = append(args, "scope", node.Scope)
	}
	if node.Metric != "" {
		args = append(args, "metric", node.Metric)
	}
	return strings.Join(args, " ")
}

// route info spec, only select important ones, too many extends
type RouteInfoSpec struct {
	Via string
	Dev string
	Mtu string
}

// info
func (info *RouteInfoSpec) String() string {
	var args []string
	if info.Via != "" {
		args = append(args, "via", info.Via)
	}
	if info.Dev != "" {
		args = append(args, "dev", info.Dev)
	}
	if info.Mtu != "" {
		args = append(args, "mtu", info.Mtu)
	}
	return strings.Join(args, " ")
}

// route
type Route struct {
	table string // ip route name // local| main | default | all | num

	// manager
	manager *Manager

	// only use
	Node RouteNodeSpec
	Info RouteInfoSpec

	// rules
	rules []*Rule
}

// do action
func (r *Route) action(action action) ([]byte, error) {
	// do action
	args := []string{"ip route", action.ToString(), r.Node.String(), r.Info.String()}
	// if table is not default
	if r.table != "" {
		args = append(args, "table", r.table)
	}
	cmd := exec.Command("/bin/sh", "-c", strings.Join(args, " "))
	logger.Debugf("[%s] begin to run command %s", cmd.String())
	return cmd.CombinedOutput()
}

// creat
func (r *Route) create() error {
	buf, err := r.action(add)
	if err != nil {
		logger.Warningf("[%s] create route failed, out: %s, err: %v", r.table, string(buf), err)
		return err
	}
	// create route success
	logger.Debugf("[%s] create route success", r.table)
	return nil
}

// remove route
func (r *Route) Remove() error {
	// del rules first
	for _, rule := range r.rules {
		buf, err := rule.Remove()
		if err != nil {
			logger.Warningf("[%s] remove rule failed, out: %s, err: %v", r.table, string(buf), err)
			return err
		}
	}
	logger.Debugf("[%s] remove all rule success", r.table)
	buf, err := r.action(del)
	if err != nil {
		logger.Warningf("[%s] create route failed, out: %s, err: %v", r.table, string(buf), err)
		return err
	}
	// create route success
	logger.Debugf("[%s] create route success", r.table)
	return nil
}

func (r *Route) CreateRule(ruleAction RuleAction, selector RuleSelector) (*Rule, error) {
	rule := &Rule{
		route:        r,
		ruleAction:   ruleAction,
		ruleSelector: selector,
	}
	buf, err := rule.create()
	if err != nil {
		logger.Warningf("[%s] create rule failed, out: %s, err: %v", r.table, string(buf), err)
		return nil, err
	}
	logger.Debugf("[%s] create rule success", r.table)
	return rule, nil
}

// rule selector
type RuleSelector struct {
	Mark       bool   // not !
	SrcPrefix  string // source
	DestPrefix string // destination
	Fwmark     string // mark
	IpProto    string // ip protocol
	SPort      string // source port
	DPort      string // destination port
}

// selector
func (s *RuleSelector) String() string {
	var args []string
	if s.Mark {
		args = append(args, "!")
	}
	if s.SrcPrefix != "" {
		args = append(args, "from", s.SrcPrefix)
	}
	if s.DestPrefix != "" {
		args = append(args, "to", s.DestPrefix)
	}
	if s.Fwmark != "" {
		args = append(args, "fwmark", s.Fwmark)
	}
	if s.IpProto != "" {
		args = append(args, "ipproto", s.IpProto)
	}
	if s.SPort != "" {
		args = append(args, "sport", s.SPort)
	}
	if s.DPort != "" {
		args = append(args, "dport", s.DPort)
	}
	return strings.Join(args, " ")
}

// rule action
type RuleAction struct {
	// Table string   this is attach with table
	Proto  string
	Nat    string
	Realms string
}

// rule action
func (a *RuleAction) String() string {
	var args []string
	if a.Proto != "" {
		args = append(args, "protocol", a.Proto)
	}
	if a.Nat != "" {
		args = append(args, "nat", a.Nat)
	}
	if a.Realms != "" {
		args = append(args, "realms", a.Realms)
	}
	return strings.Join(args, " ")
}

// rule
type Rule struct {
	route *Route // route manager

	// selector and action
	ruleSelector RuleSelector
	ruleAction   RuleAction
}

// action
func (rule *Rule) action(action action) ([]byte, error) {
	args := []string{"ip rule", action.ToString(), rule.ruleSelector.String(), rule.ruleAction.String()}
	if rule.route != nil {
		args = append(args, "table", rule.route.table)
	}
	cmd := exec.Command("/bin/sh", "-c", strings.Join(args, " "))
	logger.Debugf("[rule] run command %s", cmd.String())
	return cmd.CombinedOutput()
}

// creat
func (rule *Rule) create() ([]byte, error) {
	buf, err := rule.action(add)
	if err != nil {
		return buf, err
	}
	return nil, nil
}

func (rule *Rule) Remove() ([]byte, error) {
	buf, err := rule.action(del)
	if err != nil {
		return buf, err
	}
	return nil, nil
}
