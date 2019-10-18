package rules

import (
	"net"

	"github.com/dueckminor/go-sshtunnel/dialer"

	"github.com/dueckminor/go-sshtunnel/control"
)

type Rule struct {
	IPNet  *net.IPNet
	Dialer string
}

type RuleSet struct {
	Name  string
	Rules []Rule
}

// Marshall converts a Rule to the wire-Format (JSON)
func Marshall(rule Rule) control.Rule {
	return control.Rule{
		CIDR:   rule.IPNet.String(),
		Dialer: rule.Dialer,
	}
}

// Unmarshall converts the wire-Format (JSON) to a Rule
func Unmarshall(rule control.Rule) (Rule, error) {
	_, IPNet, err := net.ParseCIDR(rule.CIDR)

	result := Rule{
		IPNet:  IPNet,
		Dialer: rule.Dialer,
	}

	if len(result.Dialer) == 0 {
		result.Dialer = "default"
	}

	return result, err
}

func (rs *RuleSet) AddRule(rule Rule) error {
	for i, r := range rs.Rules {
		if r.IPNet.String() == rule.IPNet.String() {
			rs.Rules[i] = rule
			return nil
		}
	}
	rs.Rules = append(rs.Rules, rule)
	return nil
}

func (rs *RuleSet) ListRules() (rules []Rule, err error) {
	return rs.Rules, nil
}

var (
	defaultRuleSet = &RuleSet{
		Name: "default",
	}
)

func GetDefaultRuleSet() *RuleSet {
	return defaultRuleSet
}

func (rs *RuleSet) Dial(network, addr string) (net.Conn, error) {
	ipaddr, err := net.ResolveTCPAddr(network, addr)
	if err == nil {
		for _, rule := range rs.Rules {
			if rule.IPNet.Contains(ipaddr.IP) {
				return dialer.Dial(rule.Dialer, network, addr)
			}
		}
	}
	return net.Dial(network, addr)
}
