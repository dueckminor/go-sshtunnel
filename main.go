package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/dueckminor/gosshuttle/iptables"
	"github.com/justmao945/mallory"
)

func newSSH(user, dest string) (*mallory.SSH, error) {
	config := &mallory.Config{
		File: &mallory.ConfigFile{
			PrivateKey:   os.ExpandEnv("$HOME/.ssh/id_rsa"),
			RemoteServer: "ssh://" + user + "@" + dest + ":22"}}
	return mallory.NewSSH(config)
}

func handleConnection(ssh *mallory.SSH, conn *net.TCPConn) {
	defer conn.Close()
	ip, port, conn, err := iptables.GetOriginalDst(conn)
	if err != nil {
		L.Println("Failed to get original destination:", err)
		return
	}
	remoteAddr := ip + ":" + strconv.FormatUint(uint64(port), 10)
	L.Println("Connecting to:", remoteAddr)
	remoteConn, err := ssh.Direct.Tr.Dial("tcp", remoteAddr)
	if err != nil {
		L.Println("Failed to connect to original destination:", err)
		return
	}
	nSend, nReceived, err := forwardConnection(conn, remoteConn)

	L.Println("Send bytes:", nSend)
	L.Println("Received bytes:", nReceived)

	// if err != nil {
	// 	L.Println("Failed to transfer data to/from original destination:", err)
	// 	return
	// }
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

func start(user, dest string, networkNames ...string) {
	networks := make([]*net.IPNet, len(networkNames))

	for idx, networkName := range networkNames {
		_, network, err := net.ParseCIDR(networkName)
		if err != nil {
			panic(err)
		}
		networks[idx] = network
	}

	addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:0")
	listener, err := net.ListenTCP("tcp4", addr)
	defer listener.Close()

	addr, err = net.ResolveTCPAddr(listener.Addr().Network(), listener.Addr().String())

	fmt.Printf("Listen on port: %d", addr.Port)

	err = iptables.RedirectNetworksToPort(addr.Port, networks...)
	if err != nil {
		panic(err)
	}

	ssh, err := newSSH(user, dest)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatalln(err)
			continue
		}
		go handleConnection(ssh, conn)
	}
}

var L = log.New(os.Stdout, "sshuttle: ", log.Lshortfile|log.LstdFlags)

func init() {
	mallory.L = L
}

func main() {
	if len(os.Args) < 2 {
		panic("Usage: gosshuttle jumpbox_ip [networks...]")
	}
	if len(os.Args) == 2 {
		start(os.Getenv("USERNAME"), os.Args[1], "10.0.0.0/8")
	} else {
		start(os.Getenv("USERNAME"), os.Args[1], os.Args[2:]...)
	}
}
