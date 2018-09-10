// +build !linux

package iptables

import "net"

func GetOriginalDst(clientConn *net.TCPConn) (ipv4 string, port uint16, newTCPConn *net.TCPConn, err error) {
	return "", 0, clientConn, nil
}
