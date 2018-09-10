// +build linux

package iptables

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"
)

const SoOriginalDst = 80

func GetOriginalDst(clientConn *net.TCPConn) (string, uint16, *net.TCPConn, error) {
	if clientConn == nil {
		err := errors.New("ERR: clientConn is nil")
		return "", 0, nil, err
	}

	// test if the underlying fd is nil
	remoteAddr := clientConn.RemoteAddr()
	if remoteAddr == nil {
		err := errors.New("ERR: clientConn.fd is nil")
		return "", 0, nil, err
	}

	// net.TCPConn.File() will cause the receiver's (clientConn) socket to be placed in blocking mode.
	// The workaround is to take the File returned by .File(), do getsockopt() to get the original
	// destination, then create a new *net.TCPConn by calling net.Conn.FileConn().  The new TCPConn
	// will be in non-blocking mode.  What a pain.
	clientConnFile, err := clientConn.File()
	if err != nil {
		return "", 0, nil, err
	} else {
		clientConn.Close()
	}

	// Get original destination
	// this is the only syscall in the Golang libs that I can find that returns 16 bytes
	// Example result: &{Multiaddr:[2 0 31 144 206 190 36 45 0 0 0 0 0 0 0 0] Interface:0}
	// port starts at the 3rd byte and is 2 bytes long (31 144 = port 8080)
	// IPv4 address starts at the 5th byte, 4 bytes long (206 190 36 45)
	addr, err := syscall.GetsockoptIPv6Mreq(int(clientConnFile.Fd()), syscall.IPPROTO_IP, SoOriginalDst)
	if err != nil {
		return "", 0, nil, err
	}

	newConn, err := net.FileConn(clientConnFile)
	if err != nil {
		return "", 0, nil, err
	}
	var newTCPConn *net.TCPConn
	if _, ok := newConn.(*net.TCPConn); ok {
		newTCPConn = newConn.(*net.TCPConn)
		clientConnFile.Close()
	} else {
		err = errors.New(fmt.Sprintf("ERR: newConn is not a *net.TCPConn, instead it is: %T (%v)", newConn, newConn))
		return "", 0, nil, err
	}

	ipv4 := strconv.FormatUint(uint64(addr.Multiaddr[4]), 10) + "." +
		strconv.FormatUint(uint64(addr.Multiaddr[5]), 10) + "." +
		strconv.FormatUint(uint64(addr.Multiaddr[6]), 10) + "." +
		strconv.FormatUint(uint64(addr.Multiaddr[7]), 10)
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	return ipv4, port, newTCPConn, nil
}
