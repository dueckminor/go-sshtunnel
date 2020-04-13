package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/dueckminor/go-sshtunnel/iptables"
	"github.com/dueckminor/go-sshtunnel/sshforward"
)

// SSHTunnel handles ssh connections
type SSHTunnel struct {
	user       string
	host       string
	port       string
	timeout    int
	privateKey string
	networks   []*net.IPNet
	dns        string
}

func handleConnection(forward *sshforward.SSHForward, conn *net.TCPConn) {
	defer conn.Close()
	ip, port, conn, err := iptables.GetOriginalDst(conn)
	if err != nil {
		L.Println("Failed to get original destination:", err)
		return
	}
	remoteAddr := ip + ":" + strconv.FormatUint(uint64(port), 10)
	L.Println("Connecting to:", remoteAddr)
	remoteConn, err := forward.Dial("tcp", remoteAddr)
	if err != nil {
		L.Println("Failed to connect to original destination:", err)
		return
	}
	nSend, nReceived, err := forwardConnection(conn, remoteConn)
	if err != nil {
		L.Println("Got error:", err)
	}
	L.Println("Send bytes:", nSend)
	L.Println("Received bytes:", nReceived)
}

func forwardConnection(localConn, remoteConn net.Conn) (nSend, nReceived int64, err error) {
	done := make(chan bool)

	var errSend error
	var errReceive error

	go func() {
		defer localConn.Close()
		defer remoteConn.Close()
		nReceived, errReceive = io.Copy(localConn, remoteConn)
		done <- true
	}()
	go func() {
		defer localConn.Close()
		defer remoteConn.Close()
		nSend, errSend = io.Copy(remoteConn, localConn)
		done <- true
	}()

	_ = <-done
	_ = <-done

	if errSend != nil {
		err = errSend
	} else if errReceive != nil {
		err = errReceive
	}

	return nSend, nReceived, err
}

// Start starts an SSHTunnel on a random port on localhost
func (tunnel *SSHTunnel) Start() (err error) {

	addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:0")
	listener, err := net.ListenTCP("tcp4", addr)
	defer listener.Close()

	addr, err = net.ResolveTCPAddr(listener.Addr().Network(), listener.Addr().String())

	fmt.Printf("Listen on port: %d\n", addr.Port)

	redirectScript := &iptables.RedirectScript{}
	redirectScript.Init(addr.Port)
	redirectScript.AddNetworks(tunnel.networks...)

	if len(tunnel.dns) > 0 {
		go forwardDNS(fmt.Sprintf("127.0.0.1:%d", addr.Port), tunnel.dns)
		addrDNS, err := net.ResolveTCPAddr("tcp4", tunnel.dns)
		if err != nil {
			panic(err)
		}
		redirectScript.AddHosts(addrDNS.IP)
		redirectScript.AddDNSProxy(addr.Port)
	}

	err = redirectScript.Execute()

	if err != nil {
		panic(err)
	}

	forward, err := sshforward.NewSSHForward(tunnel.host, tunnel.port, tunnel.user, tunnel.privateKey, tunnel.timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return err
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatalln(err)
			continue
		}
		go handleConnection(forward, conn)
	}
}
