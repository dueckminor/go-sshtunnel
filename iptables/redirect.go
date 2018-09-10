package iptables

import (
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
)

func RedirectNetworksToPort(portnum int, networks ...*net.IPNet) (err error) {
	port := strconv.FormatInt(int64(portnum), 10)

	script := `#!/usr/bin/env bash
set -e
sudo iptables -t nat -N sshproxy-` + port + `
sudo iptables -t nat -F sshproxy-` + port + `
sudo iptables -t nat -I OUTPUT 1 -j sshproxy-` + port + `
sudo iptables -t nat -I PREROUTING 1 -j sshproxy-` + port + `
`
	for _, network := range networks {
		script += "sudo iptables -t nat -A sshproxy-" + port + " -j REDIRECT --dest " + network.String() + " -p tcp --to-ports " + port + "\n"
	}

	err = ioutil.WriteFile("/tmp/setup-iptables", []byte(script), 0755)
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
