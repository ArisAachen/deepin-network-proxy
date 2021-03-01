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
				match: "s",
				param: "10.20.31.189",
			},
		},
		exRules: []ExtendsRule{
			{
				match: "m",
				base: []ExtendsElem{
					{
						match: "mark",
						base: []BaseRule{
							{
								match: "mark",
								param: "1",
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
