package DBus

import (
	"errors"
	"fmt"
	"strconv"

	define "github.com/ArisAachen/deepin-network-proxy/define"
	newIptables "github.com/ArisAachen/deepin-network-proxy/new_iptables"
)

// create tables
func (mgr *proxyPrv) createTable() error {
	// start manager to init iptables and cgroups once
	mgr.manager.Start()

	// all app or global proxy has the mangle PREROUTING chain
	chain := mgr.manager.iptablesMgr.GetChain("mangle", "PREROUTING")
	if chain == nil {
		logger.Warningf("[%s] has no mangle PREROUTING chain", mgr.scope)
		return errors.New("has no mangle PREROUTING chain")
	}
	mgr.chains[0] = chain

	// get index, default append at last
	index := mgr.manager.mainChain.GetRulesCount()
	// correct index when is app proxy
	if mgr.scope == define.App {
		pos, exist := mgr.manager.mainChain.GetCreateChildIndex(define.Global.String())
		if exist {
			index = pos
		}
	}
	var mark bool
	if mgr.scope == define.Global {
		mark = true
	}

	// command line
	// iptables -t mangle -I main $1 -p tcp -m cgroup --path app.slice/global.slice -j app/global
	cpl := &newIptables.CompleteRule{
		// -j app/global
		Action: mgr.scope.String(),
		// base rules slice         -p tcp
		BaseSl: []newIptables.BaseRule{
			{
				Match: "p",
				Param: "tcp",
			},
		},
		// extends rules slice       -m cgroup --path app.slice/global.slice
		ExtendsSl: []newIptables.ExtendsRule{
			{
				Match: "m",
				Elem: newIptables.ExtendsElem{
					Match: "cgroup",
					Base:  newIptables.BaseRule{Not: mark, Match: "path", Param: mgr.controller.GetName()},
				},
			},
		},
	}
	// child chain
	childChain, err := mgr.manager.mainChain.CreateChild(mgr.scope.String(), index, cpl)
	if err != nil {
		return err
	}

	// save chain
	mgr.chains[1] = childChain

	if mgr.Proxies.DNSPort != 0 {
		chain := mgr.manager.iptablesMgr.GetChain("nat", "OUTPUT")
		if chain == nil {
			logger.Warningf("[%s] has no nat OUTPUT chain", mgr.scope)
			return errors.New("has no nat OUTPUT chain")
		}

		cpl := &newIptables.CompleteRule{
			Action: newIptables.REDIRECT,
			BaseSl: []newIptables.BaseRule{
				{
					Match: "p",
					Param: "udp",
				},
				{
					Match: "-to-ports",
					Param: "1053",
				},
			},
			ExtendsSl: []newIptables.ExtendsRule{
				{
					Match: "m",
					Elem: newIptables.ExtendsElem{
						Match: "cgroup",
						Base:  newIptables.BaseRule{Not: mark, Match: "path", Param: mgr.controller.GetName()},
					},
				},
				{
					Match: "m",
					Elem: newIptables.ExtendsElem{
						Match: "udp",
						Base:  newIptables.BaseRule{Not: mark, Match: "dport", Param: "53"},
					},
				},
			},
		}

		err := childChain.AppendRule(cpl)
		if err != nil {
			return err
		}
	}
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
	// iptables -t mangle -A PREROUTING -j TPROXY -m mark --mark $2 --on-port 8080
	protoExtends := newIptables.ExtendsRule{
		// -m
		Match: "p",
		// mark --mark $2
		Elem: newIptables.ExtendsElem{
			// mark
			Match: "tcp",
			// --mark $2
			Base: newIptables.BaseRule{
				Match: "on-port", Param: strconv.Itoa(mgr.Proxies.TPort),
			},
		},
	}
	markExtends := newIptables.ExtendsRule{
		// -m
		Match: "m",
		// mark --mark $2
		Elem: newIptables.ExtendsElem{
			// mark
			Match: "mark",
			// --mark $2
			Base: newIptables.BaseRule{
				Match: "mark", Param: strconv.Itoa(mgr.Proxies.TPort),
			},
		},
	}
	// one complete rule
	cpl = &newIptables.CompleteRule{
		// -j TPROXY
		Action: newIptables.TPROXY,
		BaseSl: nil,
		// -m mark --mark $2
		ExtendsSl: []newIptables.ExtendsRule{protoExtends, markExtends},
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
		logger.Warningf("[%s] remove self create chain failed, err: %v", mgr.scope, err)
		return err
	}

	// delete default chain from
	defChain := mgr.chains[0]
	if defChain == nil {
		logger.Warningf("[%s] default chain is nil", mgr.scope)
		return fmt.Errorf("[%s] default chain is nil", mgr.scope)
	}
	// iptables -t mangle -D PREROUTING -j TPROXY -m mark --mark $2 --on-port 8080
	protoExtends := newIptables.ExtendsRule{
		// -m
		Match: "p",
		// mark --mark $2
		Elem: newIptables.ExtendsElem{
			// mark
			Match: "tcp",
			// --mark $2
			Base: newIptables.BaseRule{
				Match: "on-port", Param: strconv.Itoa(mgr.Proxies.TPort),
			},
		},
	}
	markExtends := newIptables.ExtendsRule{
		// -m
		Match: "m",
		// mark --mark $2
		Elem: newIptables.ExtendsElem{
			// mark
			Match: "mark",
			// --mark $2
			Base: newIptables.BaseRule{
				Match: "mark", Param: strconv.Itoa(mgr.Proxies.TPort),
			},
		},
	}
	// one complete rule
	cpl := &newIptables.CompleteRule{
		// -j TPROXY
		Action: newIptables.TPROXY,
		BaseSl: nil,
		// -m mark --mark $2
		ExtendsSl: []newIptables.ExtendsRule{protoExtends, markExtends},
	}
	err = defChain.DelRule(cpl)
	if err != nil {
		logger.Warningf("[%s] delete rule failed, err: %v", mgr.scope, err)
		return err
	}
	return nil
}

// release controller
func (mgr *proxyPrv) releaseController() error {
	return mgr.controller.ReleaseAll()
}
