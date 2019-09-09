package sshtunnel

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/dueckminor/go-sshtunnel/dnsproxy"
	"github.com/dueckminor/go-sshtunnel/iptables"
	"github.com/dueckminor/go-sshtunnel/logger"
	"github.com/dueckminor/go-sshtunnel/sshforward"
)

type SSHTunnel struct {
	User       string
	Host       string
	Port       string
	Timeout    int
	PrivateKey string
	Networks   []*net.IPNet
	DNS        string
}

func handleConnection(forward *sshforward.SSHForward, conn *net.TCPConn) {
	defer conn.Close()
	ip, port, conn, err := iptables.GetOriginalDst(conn)
	if err != nil {
		logger.L.Println("Failed to get original destination:", err)
		return
	}
	remoteAddr := ip + ":" + strconv.FormatUint(uint64(port), 10)
	logger.L.Println("Connecting to:", remoteAddr)
	remoteConn, err := forward.Dial("tcp", remoteAddr)
	if err != nil {
		logger.L.Println("Failed to connect to original destination:", err)
		return
	}
	nSend, nReceived, err := forwardConnection(conn, remoteConn)

	logger.L.Println("Send bytes:", nSend)
	logger.L.Println("Received bytes:", nReceived)
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

func (tunnel *SSHTunnel) Start() (err error) {

	addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:0")
	listener, err := net.ListenTCP("tcp4", addr)
	defer listener.Close()

	addr, err = net.ResolveTCPAddr(listener.Addr().Network(), listener.Addr().String())

	fmt.Printf("Listen on port: %d\n", addr.Port)

	redirectScript := &iptables.RedirectScript{}
	redirectScript.Init(addr.Port)
	redirectScript.AddNetworks(tunnel.Networks...)

	if len(tunnel.DNS) > 0 {
		go dnsproxy.ForwardDNS(fmt.Sprintf("127.0.0.1:%d", addr.Port), tunnel.DNS)
		addrDNS, err := net.ResolveTCPAddr("tcp4", tunnel.DNS)
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

	forward, err := sshforward.NewSSHForward(tunnel.Host, tunnel.Port, tunnel.User, tunnel.PrivateKey, tunnel.Timeout)
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
