package rules

import (
	"net"

	"github.com/dueckminor/go-sshtunnel/dialer"

	"github.com/dueckminor/go-sshtunnel/control"
)

// A Rule binds a CIDR range to a dialer
type Rule struct {
	IPNet  *net.IPNet
	Dialer string
}

// A RuleSet is a named set of Rules
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

// UnMarshall converts the wire-Format (JSON) to a Rule
func UnMarshall(rule control.Rule) (Rule, error) {
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

// AddRule adds a single rule to a RuleSet. If the CIDR range is already
// part if the RuleSet, the existing rule will be replaced
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

// ListRules returns all Rules
func (rs *RuleSet) ListRules() (rules []Rule, err error) {
	return rs.Rules, nil
}

var (
	defaultRuleSet = &RuleSet{
		Name: "default",
	}
)

// GetDefaultRuleSet returns the default RuleSet
func GetDefaultRuleSet() *RuleSet {
	return defaultRuleSet
}

// Dial uses the dialer of the first matching rule to establish a network connection
func (rs *RuleSet) Dial(network, addr string) (net.Conn, error) {
	ipAddr, err := net.ResolveTCPAddr(network, addr)
	if err == nil {
		for _, rule := range rs.Rules {
			if rule.IPNet.Contains(ipAddr.IP) {
				return dialer.Dial(rule.Dialer, network, addr)
			}
		}
	}
	return net.Dial(network, addr)
}
