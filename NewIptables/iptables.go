package NewIptables

import (
	"errors"
	"os/exec"
	"reflect"
	"strconv"
	"strings"

	com "github.com/DeepinProxy/Com"
)

// tables
type Table struct {
	Name   string // raw mangle nat filter
	chains map[string]*Chain
}

// run iptables command
func (t *Table) runCommand(operation Operation, chain *Chain, index int, cpl *CompleteRule) error {
	// run command
	args := []string{"iptables", "-t", t.Name, "-" + operation.ToString(), chain.Name}
	// add index
	if index != 0 && operation == Insert {
		args = append(args, strconv.Itoa(index))
	}
	// add one complete rule
	if cpl != nil {
		args = append(args, cpl.String())
	}
	cmd := exec.Command("/bin/sh", "-c", strings.Join(args, " "))
	logger.Debugf("[%s] begin to run begin to run command: %s", t.Name, cmd.String())
	buf, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warningf("[%s] run command failed, out: %s, err:%v", t.Name, string(buf), err)
		return err
	}
	logger.Debugf("[%s] run command success", t.Name)
	return nil
}

// check if chain exist
func (t *Table) getChain(name string) *Chain {
	chain, ok := t.chains[name]
	if !ok {
		logger.Warningf("[%s] chain %s not exist", t.Name, name)
		return nil
	}
	logger.Debugf("[%s] chain %s found", t.Name, name)
	return chain
}

// chain
type Chain struct {
	// chain name
	Name string // PREROUTING,INPUT,FORWARD,OUTPUT,POSTROUTING or self define chain

	// table
	table *Table

	// parent chain
	parent *Chain
	// children chain
	children map[string]*Chain

	cplRuleSl []*CompleteRule
}

// save parent
func (c *Chain) setParent(parent *Chain) {
	c.parent = parent
}

// check index valid
func (c *Chain) indexValid(index int) bool {
	return len(c.cplRuleSl) >= index
}

// create child chain
func (c *Chain) CreateChild(name string, index int, cpl *CompleteRule) (*Chain, error) {
	// create child
	child := &Chain{
		Name:     name,
		table:    c.table, // the same table with parent
		parent:   c,       // set this as parent
		children: make(map[string]*Chain),
	}
	// create chain
	err := c.table.runCommand(New, child, 0, nil)
	if err != nil {
		logger.Warningf("[%s] create child %s failed, err: %v", c.table.Name, name, err)
		return nil, err
	}
	logger.Debugf("[%s] create chain %s success", c.table.Name, name)
	// start to attach
	err = c.InsertRule(Insert, index, cpl)
	if err != nil {
		logger.Warningf("[%s] chain %s attach child %s failed, err: %v", c.table.Name, c.Name, name, err)
		return nil, err
	}
	// add to table
	c.table.chains[name] = child
	// add to child
	c.children[name] = child
	logger.Debugf("[%s] chain %s create child %s success", c.table.Name, c.Name, name)
	// return handler
	return child, nil
}

// current rule count
func (c *Chain) GetRulesCount() int {
	return len(c.cplRuleSl)
}

// current children chain
func (c *Chain) GetChildrenCount() int {
	return len(c.children)
}

// current create child index
func (c *Chain) GetCreateChildIndex(name string) (int, bool) {
	// search all rule
	for index, rule := range c.cplRuleSl {
		if strings.Contains(rule.String(), strings.Join([]string{"-j", name}, " ")) {
			logger.Debugf("[%s] chain %s has child %s in %v", c.table.Name, c.Name, name, index)
			return index, true
		}
	}
	logger.Debugf("[%s] chain %s has not child %s", c.table.Name, c.Name, name)
	return 0, false
}

// remove self
func (c *Chain) Remove() error {
	// delete self from parent first
	if c.parent != nil {
		err := c.parent.DelChild(c)
		if err != nil {
			return err
		}
	}
	// flush self   sudo iptables -t mangle -F OUTPUT
	err := c.Clear()
	if err != nil {
		return err
	}
	// remove self from table
	err = c.table.runCommand(Remove, c, 0, nil)
	return err
}

// clear all chain
func (c *Chain) Clear() error {
	for _, child := range c.children {
		err := child.Remove()
		if err != nil {
			logger.Warningf("[%s] chain %s remove child chain %s failed, err: %v", c.table.Name, c.Name, child.Name, err)
			continue
		}
		logger.Debugf("[%s] chain %s remove child chain %s success", c.table.Name, c.Name, child.Name)
	}
	// clear self chain
	err := c.table.runCommand(Flush, c, 0, nil)
	if err != nil {
		logger.Warningf("[%s] chain %s flush failed", c.table.Name, c.Name, err)
		return err
	}
	// reset all rule
	c.cplRuleSl = []*CompleteRule{}
	logger.Debugf("[%s] chain %s flush success", c.table.Name, c.Name)
	return nil
}

// delete child from self
func (c *Chain) DelChild(child *Chain) error {
	var childName string
	// check if chain exist
	for name, chain := range c.children {
		// find child
		if chain == child {
			childName = name
		}
	}
	// check if child name is nil
	if childName == "" {
		logger.Warningf("[%s] chain %s has not child %s", c.table.Name, c.Name, child.Name)
		return nil
	}
	logger.Debugf("[%s] chain %s has child %s, begin to delete", c.table.Name, c.Name, child.Name)
	if index, exist := c.GetCreateChildIndex(child.Name); exist {
		return c.DelIndexRule(index)
	}
	return nil
}

// add rule
func (c *Chain) AddRule(cpl *CompleteRule) error {
	return c.InsertRule(Insert, 0, cpl)
}

// append rule at last
func (c *Chain) AppendRule(cpl *CompleteRule) error {
	// check if already exist
	if c.ExistRule(cpl) {
		return nil
	}
	// clear self chain
	err := c.table.runCommand(Append, c, 0, cpl)
	if err != nil {
		logger.Warningf("[%s] chain %s flush failed, err: %v", c.table.Name, c.Name, err)
		return err
	}
	c.cplRuleSl = append(c.cplRuleSl, cpl)
	return nil
}

// insert rule
func (c *Chain) InsertRule(operation Operation, index int, cpl *CompleteRule) error {
	if !c.indexValid(index) {
		logger.Warningf("[%s] chain %s add rule failed, index invalid", c.table.Name, c.Name)
		return errors.New("index invalid")
	}
	// check if already exist
	if c.ExistRule(cpl) {
		return nil
	}
	// clear self chain
	err := c.table.runCommand(operation, c, index+1, cpl)
	if err != nil {
		logger.Warningf("[%s] chain %s flush failed", c.table.Name, c.Name, err)
		return err
	}
	logger.Debugf("[%s] chain %s insert success", c.table.Name, c.Name)
	ifc, update, err := com.MegaInsert(c.cplRuleSl, cpl, index)
	if err != nil {
		logger.Warningf("[%s] inset failed, err: %v", c.table.Name, err)
		return err
	}
	if !update {
		return nil
	}
	temp, ok := ifc.([]*CompleteRule)
	if !ok {
		return nil
	}
	c.cplRuleSl = temp
	return nil
}

// check if rule exist
func (c *Chain) ExistRule(cpl *CompleteRule) bool {
	for _, rule := range c.cplRuleSl {
		if reflect.DeepEqual(rule, cpl) {
			logger.Debugf("[%s] chain %s exist rule %s", c.table.Name, c.Name, cpl.String())
			return true
		}
	}
	logger.Debugf("[%s] chain %s dont exist rule %s", c.table.Name, c.Name, cpl.String())
	return false
}

// del rule
func (c *Chain) DelRule(cpl *CompleteRule) error {
	// check if rule exist
	if !c.ExistRule(cpl) {
		return nil
	}
	// clear self chain
	err := c.table.runCommand(Delete, c, 0, cpl)
	if err != nil {
		logger.Warningf("[%s] chain %s del failed", c.table.Name, c.Name, err)
		return err
	}
	// delete slice
	ifc, update, err := com.MegaDel(c.cplRuleSl, cpl)
	if err != nil {
		logger.Warningf("[%s] del failed, err: %v", c.table.Name, err)
		return err
	}
	if !update {
		return nil
	}
	temp, ok := ifc.([]*CompleteRule)
	if !ok {
		return nil
	}
	c.cplRuleSl = temp
	return nil
}

// get rule index
func (c *Chain) GetIndexRule(index int) *CompleteRule {
	if index >= len(c.cplRuleSl) {
		return nil
	}
	return c.cplRuleSl[index]
}

// del rule index
func (c *Chain) DelIndexRule(index int) error {
	rule := c.GetIndexRule(index)
	if rule == nil {
		return errors.New("index invalid")
	}
	err := c.DelRule(rule)
	return err
}
