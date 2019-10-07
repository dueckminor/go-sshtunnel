package commands

import (
	"flag"
	"fmt"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("list-rules", (&cmdListRules{}).Init())
	RegisterCommand("add-rule", (&cmdAddRule{}).Init())
}

type cmdListRules struct {
	flags *flag.FlagSet
}

func (cmd *cmdListRules) Init() *cmdListRules {
	cmd.flags = flag.NewFlagSet("list-rules", flag.ContinueOnError)
	defUsage := cmd.flags.Usage
	cmd.flags.Usage = func() {
		defUsage()
	}
	return cmd
}

func (cmd *cmdListRules) Execute(args ...string) error {
	cmd.flags.Parse(args)
	rules, err := control.Client().ListRules()
	if err != nil {
		return err
	}
	for _, rule := range rules {
		fmt.Println(rule.CIDR)
	}
	return nil
}

type cmdAddRule struct {
	flags  *flag.FlagSet
	dialer string
}

func (cmd *cmdAddRule) Init() *cmdAddRule {
	cmd.flags = flag.NewFlagSet("add-rule", flag.ContinueOnError)
	cmd.flags.StringVar(&cmd.dialer, "dialer", "default", "the dialer which shall be used if the rule matches")
	cmd.flags.Usage = func() {
		fmt.Println("\nUsage: sshtunnel add-rule [options] cidr...")
		cmd.flags.PrintDefaults()
	}
	return cmd
}

func (cmd *cmdAddRule) Execute(args ...string) error {
	cmd.flags.Parse(args)

	if 0 == cmd.flags.NArg() {
		cmd.flags.Usage()
		return nil
	}

	for _, cidr := range cmd.flags.Args() {
		rule := control.Rule{
			CIDR:   cidr,
			Dialer: cmd.dialer,
		}
		err := control.Client().AddRule(rule)
		if err != nil {
			return err
		}
	}
	return nil
}
