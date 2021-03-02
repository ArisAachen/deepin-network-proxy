package Iptables

import (
	"log"
	"testing"
)

func TestRuleCommand_CombinedOutput(t *testing.T) {
	contain := containRule{
		action: ACCEPT,
		bsRules: []BaseRule{
			{
				Match: "s",
				Param: "10.20.31.189",
			},
		},
		exRules: []ExtendsRule{
			{
				Match: "m",
				Base: []ExtendsElem{
					{
						Match: "mark",
						Base: []BaseRule{
							{
								Match: "mark",
								Param: "1",
							},
						},
					},
				},
			},
		},
	}

	rCmd := RuleCommand{
		soft:      "iptables",
		table:     "mangle",
		operation: Append,
		chain:     "OUTPUT",
		contain:   contain,
	}
	log.Println(rCmd)
}
