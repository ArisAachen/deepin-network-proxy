package Iptables

import (
	"log"
	"testing"
)

func TestRuleCommand_CombinedOutput(t *testing.T) {
	contain := containRule{
		action: ACCEPT,
		bsRules: []baseRule{
			{
				match: "s",
				param: "10.20.31.189",
			},
		},
		exRules: []extendsRule{
			{
				match: "m",
				base: []extendsElem{
					{
						match: "mark",
						base: []baseRule{
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
