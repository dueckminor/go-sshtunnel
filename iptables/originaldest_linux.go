// +build linux

package iptables

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"
)

// get the original destination for the socket when redirect by linux iptables
// refer to https://raw.githubusercontent.com/missdeer/avege/master/src/inbound/redir/redir_iptables.go

const SO_ORIGINAL_DST = 80

func GetOriginalDst(clientConn *net.TCPConn) (ipv4 string, port uint16, newTCPConn *net.TCPConn, err error) {
	if clientConn == nil {
		//log.Debugf("copy(): oops, dst is nil!")
		err = errors.New("ERR: clientConn is nil")
		return
	}

	// test if the underlying fd is nil
	remoteAddr := clientConn.RemoteAddr()
	if remoteAddr == nil {
		//log.Debugf("getOriginalDst(): oops, clientConn.fd is nil!")
		err = errors.New("ERR: clientConn.fd is nil")
		return
	}

	//srcipport := fmt.Sprintf("%v", clientConn.RemoteAddr())

	newTCPConn = nil
	// net.TCPConn.File() will cause the receiver's (clientConn) socket to be placed in blocking mode.
	// The workaround is to take the File returned by .File(), do getsockopt() to get the original
	// destination, then create a new *net.TCPConn by calling net.Conn.FileConn().  The new TCPConn
	// will be in non-blocking mode.  What a pain.
	clientConnFile, err := clientConn.File()
	if err != nil {
		//log.Infof("GETORIGINALDST|%v->?->FAILEDTOBEDETERMINED|ERR: could not get a copy of the client connection's file object", srcipport)
		return
	} else {
		clientConn.Close()
	}

	// Get original destination
	// this is the only syscall in the Golang libs that I can find that returns 16 bytes
	// Example result: &{Multiaddr:[2 0 31 144 206 190 36 45 0 0 0 0 0 0 0 0] Interface:0}
	// port starts at the 3rd byte and is 2 bytes long (31 144 = port 8080)
	// IPv4 address starts at the 5th byte, 4 bytes long (206 190 36 45)
	addr, err := syscall.GetsockoptIPv6Mreq(int(clientConnFile.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	//log.Debugf("getOriginalDst(): SO_ORIGINAL_DST=%+v\n", addr)
	if err != nil {
		//log.Infof("GETORIGINALDST|%v->?->FAILEDTOBEDETERMINED|ERR: getsocketopt(SO_ORIGINAL_DST) failed: %v", srcipport, err)
		return
	}
	newConn, err := net.FileConn(clientConnFile)
	if err != nil {
		//log.Infof("GETORIGINALDST|%v->?->%v|ERR: could not create a FileConn fron clientConnFile=%+v: %v", srcipport, addr, clientConnFile, err)
		return
	}
	if _, ok := newConn.(*net.TCPConn); ok {
		newTCPConn = newConn.(*net.TCPConn)
		clientConnFile.Close()
	} else {
		errmsg := fmt.Sprintf("ERR: newConn is not a *net.TCPConn, instead it is: %T (%v)", newConn, newConn)
		//log.Infof("GETORIGINALDST|%v->?->%v|%s", srcipport, addr, errmsg)
		err = errors.New(errmsg)
		return
	}

	ipv4 = strconv.FormatUint(uint64(addr.Multiaddr[4]), 10) + "." +
		strconv.FormatUint(uint64(addr.Multiaddr[5]), 10) + "." +
		strconv.FormatUint(uint64(addr.Multiaddr[6]), 10) + "." +
		strconv.FormatUint(uint64(addr.Multiaddr[7]), 10)
	port = uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	return
}
