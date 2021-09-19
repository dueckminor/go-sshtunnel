package proxy

import (
	"fmt"
	"net"
	"os"

	"github.com/dueckminor/go-sshtunnel/dialer"
	"github.com/miekg/dns"
)

func init() {
	RegisterProxyFactory("dns", newDNSProxy)
}

type dnsProxy struct {
	Dialer dialer.Dialer
	Port   int
}

func (proxy *dnsProxy) GetPort() int {
	return proxy.Port
}

func (proxy *dnsProxy) SetDialer(dialer dialer.Dialer) {
	proxy.Dialer = dialer
}

func getFreeUDPPort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.LocalAddr().(*net.UDPAddr).Port, nil
}

func makeTargetAddr(parameters string) (target string, err error) {
	host, port, err := net.SplitHostPort(parameters)
	if (err != nil) && parameters != "" {
		return "", err
	}
	if host == "" {
		host = "127.0.0.53"
	}
	if port == "" {
		port = "53"
	}
	return host + ":" + port, nil
}

func newDNSProxy(parameters string) (Proxy, error) {
	port, err := getFreeUDPPort()
	if err != nil {
		return nil, err
	}

	return NewDNSProxy(nil, port, parameters)
}

func NewDNSProxy(dialer dialer.Dialer, port int, parameters string) (Proxy, error) {
	target, err := makeTargetAddr(parameters)
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(os.Stderr, "newDNSProxy:", target)

	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)

	proxy := &dnsProxy{}
	proxy.Port = port
	proxy.Dialer = dialer
	go forwardDNS(listenAddr, target)
	return proxy, nil
}

func forwardDNS(listenAddr, targetAddr string) error {
	fmt.Printf("Forward DNS requests to: %s\n", targetAddr)

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("panic occurred:", err)
			}
		}()
		switch r.Opcode {
		case dns.OpcodeQuery:
			dnsClient := new(dns.Client)
			dnsClient.Net = "tcp"
			fmt.Println("----- REQUEST -----")
			fmt.Println(r)
			response, _, err := dnsClient.Exchange(r, targetAddr)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("----- RESPONSE -----")
			fmt.Println(response)
			w.WriteMsg(response)
		}
	})
	server := &dns.Server{Addr: listenAddr, Net: "udp"}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
	return nil
}

// cSpell: ignore miekg
