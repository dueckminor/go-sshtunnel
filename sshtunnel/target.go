package sshtunnel

import "net"

// Target binds a network to a tunnel
type Target struct {
	Network *net.IPNet
	Tunnel  string
}

// Targets allows to find a tunnel for a network
type Targets struct {
	Targets []Target
}

// AddTarget adds a target
func (targets *Targets) AddTarget(cidr string, Tunnel string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	targets.Targets = append(targets.Targets, Target{network, Tunnel})
	return nil
}
