package commands

import (
	"fmt"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("list-rules", cmdListRules{})
	RegisterCommand("add-rule", cmdAddRule{})
}

type cmdListRules struct{}
type cmdAddRule struct{}

func (cmdListRules) Execute(args ...string) error {
	rules, err := control.Client().ListRules()
	if err != nil {
		return err
	}
	for _, rule := range rules {
		fmt.Println(rule.CIDR)
	}
	return nil
}

func (cmdAddRule) Execute(args ...string) error {
	rule := control.Rule{
		CIDR:   args[0],
		Dialer: "default",
	}
	return control.Client().AddRule(rule)
}
