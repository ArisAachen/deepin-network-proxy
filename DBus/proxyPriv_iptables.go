package DBus

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	cgroup "github.com/DeepinProxy/CGroups"
	newIptables "github.com/DeepinProxy/NewIptables"
)

// init iptables, may include read origin iptables later
func (mgr *proxyPrv) initNewIptables() error {
	// init once
	if onceNewTb == nil {
		// reset once to init func once
		onceNewTb = new(sync.Once)
	}

	onceNewTb.Do(func() {
		logger.Debug("once new iptables")
		// init  iptables manager once
		allNewIptables = newIptables.NewManager()
		// init once
		allNewIptables.Init()
		// create mangle output
		outputChain := allNewIptables.GetChain("mangle", "OUTPUT")
		// create child chain under output chain
		mainChain, err := outputChain.CreateChild("MainEntry", 0, &newIptables.CompleteRule{Action: "MainEntry"})
		if err != nil {
			logger.Warningf("[%s] create main chain failed, err: %v", mgr.scope, err)
			return
		}
		// mainChain add default rule
		// iptables -t mangle -A All_Entry -m cgroup --path main.slice -j RETURN
		extends := newIptables.ExtendsRule{
			// -m
			Match: "m",
			// cgroup --path main.slice
			Elem: newIptables.ExtendsElem{
				// cgroup
				Match: "cgroup",
				// --path main.slice
				Base: newIptables.BaseRule{Match: "path", Param: cgroup.MainGRP + ".slice"},
			},
		}
		// one complete rule
		cpl := &newIptables.CompleteRule{
			// -j RETURN
			Action: newIptables.RETURN,
			BaseSl: nil,
			// -m cgroup --path main.slice -j RETURN
			ExtendsSl: []newIptables.ExtendsRule{extends},
		}
		// append rule
		err = mainChain.AppendRule(cpl)
		return
	})
	// all app or global proxy has the mangle PREROUTING chain
	chain := allNewIptables.GetChain("mangle", "PREROUTING")
	if chain == nil {
		logger.Warningf("[%s] has no mangle PREROUTING chain", mgr.scope)
		return errors.New("has no mangle PREROUTING chain")
	}
	mgr.chains[0] = chain
	return nil
}

// add rule at App_Proxy or mangle OUTPUT
func (mgr *proxyPrv) appendRule() error {
	// get chain
	selfChain := mgr.chains[1]
	if selfChain == nil {
		logger.Warningf("[%s] cant add rule, chain is nil", mgr.scope)
		return errors.New("chain is nil")
	}
	// iptables -t mangle -A App_Proxy -j MARK --set-mark $2
	base := newIptables.BaseRule{
		Match: "-set-mark",
		Param: strconv.Itoa(mgr.Proxies.TPort),
	}
	// one complete rule
	cpl := &newIptables.CompleteRule{
		// -j MARK
		Action: newIptables.MARK,
		// --set-mark $2
		BaseSl: []newIptables.BaseRule{base},
	}
	// append
	err := selfChain.AppendRule(cpl)
	if err != nil {
		return err
	}

	// default chain
	defChain := mgr.chains[0]
	if defChain == nil {
		logger.Warningf("[%s] cant add rule, chain is nil", mgr.scope)
		return errors.New("chain is nil")
	}
	// iptables -t mangle -A PREROUTING -j TPROXY -m mark --mark $2
	extends := newIptables.ExtendsRule{
		// -m
		Match: "m",
		// mark --mark $2
		Elem: newIptables.ExtendsElem{
			// mark
			Match: "mark",
			// --mark $2
			Base: newIptables.BaseRule{Match: "mark", Param: strconv.Itoa(mgr.Proxies.TPort)},
		},
	}
	// one complete rule
	cpl = &newIptables.CompleteRule{
		// -j TPROXY
		Action: newIptables.TPROXY,
		BaseSl: nil,
		// -m mark --mark $2
		ExtendsSl: []newIptables.ExtendsRule{extends},
	}
	// append
	err = defChain.AppendRule(cpl)
	if err != nil {
		return err
	}
	return nil
}

// delete chain and remove from parent
func (mgr *proxyPrv) releaseRule() error {
	// clear self chain
	selfChain := mgr.chains[1]
	if selfChain == nil {
		logger.Warningf("[%s] self create chain is nil", mgr.scope)
		return fmt.Errorf("[%s] self create chain is nil", mgr.scope)
	}
	err := selfChain.Remove()
	if err != nil {
		logger.Warningf("[%s] remove self create chain failed, err: %v", err)
		return err
	}

	// delete default chain from
	defChain := mgr.chains[0]
	if defChain == nil {
		logger.Warningf("[%s] default chain is nil", mgr.scope)
		return fmt.Errorf("[%s] default chain is nil", mgr.scope)
	}
	// iptables -t mangle -D PREROUTING -j TPROXY -m mark --mark $2
	extends := newIptables.ExtendsRule{
		// -m
		Match: "m",
		// mark --mark $2
		Elem: newIptables.ExtendsElem{
			// mark
			Match: "mark",
			// --mark $2
			Base: newIptables.BaseRule{Match: "mark", Param: strconv.Itoa(mgr.Proxies.TPort)},
		},
	}
	// one complete rule
	cpl := &newIptables.CompleteRule{
		// -j TPROXY
		Action: newIptables.TPROXY,
		BaseSl: nil,
		// -m mark --mark $2
		ExtendsSl: []newIptables.ExtendsRule{extends},
	}
	err = defChain.DelRule(cpl)
	if err != nil {
		logger.Warningf("[%s] delete rule failed, err: %v", mgr.scope, err)
		return err
	}

	// get main chain
	mainChain := allNewIptables.GetChain("mangle", "MainEntry")
	if mainChain == nil {
		return nil
	}
	// check if has children, if not delete self
	if mainChain.GetChildrenCount() == 0 {
		// remove self
		err = mainChain.Remove()
		if err != nil {
			logger.Warningf("[%s] remove main chain failed, err: %v", mgr.scope, err)
			return err
		}
		// reset
		onceNewTb = nil
		allNewIptables = nil
	}
	return nil
}
