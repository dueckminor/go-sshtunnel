package proxy

import (
	"io"
	"log"
	"net"
	"strconv"

	"github.com/dueckminor/go-sshtunnel/originaldest"
	"github.com/dueckminor/go-sshtunnel/rules"

	"github.com/dueckminor/go-sshtunnel/dialer"
	"github.com/dueckminor/go-sshtunnel/logger"
)

type tcpProxy struct {
	Dialer dialer.Dialer
	Port   int
}

func init() {
	RegisterProxyFactory("tcp", newTCPProxy)
}

func newTCPProxy(parameters string) (Proxy, error) {
	proxy := &tcpProxy{}
	var err error

	proxy.Dialer = rules.GetDefaultRuleSet()

	port := 0
	if len(parameters) > 0 {
		port, err = strconv.Atoi(parameters)
		if err != nil {
			return nil, err
		}
	}

	err = proxy.start(port)
	if err != nil {
		return nil, err
	}

	return proxy, nil
}

func (proxy *tcpProxy) GetPort() int {
	return proxy.Port
}

func (proxy *tcpProxy) SetDialer(dialer dialer.Dialer) {
	proxy.Dialer = dialer
}

func handleConnection(dialer dialer.Dialer, conn *net.TCPConn) {
	defer conn.Close()
	ip, port, conn, err := originaldest.GetOriginalDst(conn)
	if err != nil {
		logger.L.Println("Failed to get original destination:", err)
		return
	}
	remoteAddr := ip + ":" + strconv.FormatUint(uint64(port), 10)
	logger.L.Println("Connecting to:", remoteAddr)
	remoteConn, err := dialer.Dial("tcp", remoteAddr)
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

func (proxy *tcpProxy) start(port int) (err error) {
	listener, port, err := createTCPListener(port)
	if err != nil {
		return err
	}
	proxy.Port = port

	go func() {
		defer listener.Close()

		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Fatalln(err)
				continue
			}
			go handleConnection(proxy.Dialer, conn)
		}
	}()

	return nil
}
