package iptables

import (
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
)

type RedirectScript struct {
	script string
	table  string
	port   string
}

// Init prepares the redirect-script
func (rs *RedirectScript) Init(portnum int) {
	rs.port = strconv.FormatInt(int64(portnum), 10)
	rs.table = "sshproxy-" + rs.port

	rs.script = "#!/usr/bin/env bash\n"
	rs.script += "\n"
	rs.script += "set -e"
	rs.script += "\n"
	rs.script += "sudo iptables-save | grep -v sshproxy | sudo iptables-restore\n"
	rs.script += "sudo iptables -t nat -N " + rs.table + "\n"
	rs.script += "sudo iptables -t nat -F " + rs.table + "\n"
	rs.script += "sudo iptables -t nat -I OUTPUT 1 -j " + rs.table + "\n"
	rs.script += "sudo iptables -t nat -I PREROUTING 1 -j " + rs.table + "\n"
}

// AddNetworks add a network to the script
func (rs *RedirectScript) AddNetworks(networks ...*net.IPNet) {
	for _, network := range networks {
		rs.script += "sudo iptables -t nat -A " + rs.table + " -j REDIRECT --dest " + network.String() + " -p tcp --to-ports " + rs.port + "\n"
	}
}

// AddHosts add a network to the script
func (rs *RedirectScript) AddHosts(ips ...net.IP) {
	for _, ip := range ips {
		rs.script += "sudo iptables -t nat -A " + rs.table + " -j REDIRECT --dest " + ip.String() + " -p tcp --to-ports " + rs.port + "\n"
	}
}

// AddDNSProxy add a dns-proxy to the script
func (rs *RedirectScript) AddDNSProxy(dnsPortnum int) {
	rs.script += "sudo iptables -t nat -A " + rs.table + " -p udp --dport 53 -j REDIRECT --to-ports " + strconv.FormatInt(int64(dnsPortnum), 10) + "\n"
}

// Execute executes the script
func (rs *RedirectScript) Execute() (err error) {
	err = ioutil.WriteFile("/tmp/setup-iptables", []byte(rs.script), 0755)
	if err != nil {
		return err
	}

	cmd := exec.Command("/tmp/setup-iptables")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	return err
}
