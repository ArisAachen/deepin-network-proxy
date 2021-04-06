package NewIptables

import "strings"

// define operation
type Operation int

const (
	Append Operation = iota
	Insert
	New
	Delete
	Remove
	Policy
	Flush
)

func (a Operation) ToString() string {
	switch a {
	case Append:
		return "A"
	case Insert:
		return "I"
	case New:
		return "N"
	case Delete:
		return "D"
	case Remove:
		return "X"
	case Policy:
		return "P"
	case Flush:
		return "F"
	default:
		return ""
	}
}

// action
const (
	ACCEPT   = "ACCEPT"
	DROP     = "DROP"
	RETURN   = "RETURN"
	QUEUE    = "QUEUE"
	REDIRECT = "REDIRECT"
	TPROXY   = "TPROXY"
	MARK     = "MARK"
)

// base rule
type BaseRule struct {
	Not   bool   // !
	Match string // -s
	Param string // 1111.2222.3333.4444
}

// make string  -s 1111.2222.3333.4444
func (bs *BaseRule) String() string {
	var sl []string
	// if mark as false
	if bs.Not {
		sl = append(sl, "!")
	}
	sl = append(sl, "-"+bs.Match, bs.Param)
	return strings.Join(sl, " ")
}

// extends elem
type ExtendsElem struct {
	Match string   // mark
	Base  BaseRule // --mark 1
}

// make string    mark --mark 1
func (elem *ExtendsElem) String() string {
	sl := []string{elem.Match}
	if elem.Base.Not {
		sl = append(sl, "!")
	}
	sl = append(sl, "--"+elem.Base.Match, elem.Base.Param)
	return strings.Join(sl, " ")
}

// extends rule
type ExtendsRule struct {
	Match string      // -m
	Elem  ExtendsElem // mark --mark 1
}

// make string   -m mark --mark 1
func (ex *ExtendsRule) String() string {
	sl := []string{"-" + ex.Match, ex.Elem.String()}
	return strings.Join(sl, " ")
}

// one complete rule
type CompleteRule struct {
	Action    string
	BaseSl    []BaseRule
	ExtendsSl []ExtendsRule
}

// make string        -j ACCEPT -s 1111.2222.3333.4444 -m mark --mark 1
func (cpl *CompleteRule) String() string {
	// action
	sl := []string{"-j", cpl.Action}
	// base rules
	for _, base := range cpl.BaseSl {
		sl = append(sl, base.String())
	}
	// extends rules
	for _, extends := range cpl.ExtendsSl {
		sl = append(sl, extends.String())
	}
	return strings.Join(sl, " ")
}
