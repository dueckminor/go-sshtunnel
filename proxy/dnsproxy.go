package proxy

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/dueckminor/go-sshtunnel/dialer"
	"github.com/miekg/dns"
)

var timeFormat = "2006-01-02 15:04:05"

var dnsTarget = ""

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

	listenAddr := fmt.Sprintf(":%d", port)

	proxy := &dnsProxy{}
	proxy.Port = port
	dnsTarget = target
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

		msgSize := uint16(dns.MinMsgSize)
		//check if the client accepts a different udp message size
		opt := r.IsEdns0()
		if opt != nil && opt.UDPSize() >= dns.MinMsgSize {
			msgSize = opt.UDPSize()
		}

		switch r.Opcode {
		case dns.OpcodeQuery:

			dnsClient := new(dns.Client)
			dnsClient.WriteTimeout = 10 * time.Second
			dnsClient.ReadTimeout = 10 * time.Second
			dnsClient.Net = "tcp"

			fmt.Println("----- REQUEST -----")
			fmt.Println("Time:", time.Now().Format(timeFormat))
			fmt.Println("LocalAddr:", w.LocalAddr())
			fmt.Println("RemoteAddr:", w.RemoteAddr())
			fmt.Println("OPT PackageSize:", msgSize)
			fmt.Println(r)

			response, runtime, err := dnsClient.Exchange(r, targetAddr)
			if err != nil {
				fmt.Println("----- ERROR -----")
				fmt.Println("Time:", time.Now().Format(timeFormat))
				fmt.Println(err)
			}

			fmt.Println("----- RESPONSE -----")
			fmt.Println("Time:", time.Now().Format(timeFormat))
			fmt.Println("Runtime:", runtime)
			fmt.Println("LocalAddr:", w.LocalAddr())
			fmt.Println("RemoteAddr:", w.RemoteAddr())
			fmt.Println("Response:", response)

			// as we get the response via TCP and have to send it to our client
			// via UDP, the message size MUST NOT exceed 512 bytes.
			// In case we get w longer response, we have to mark it as truncated
			// and remove as many records as needed to get a message which
			// is not longer than 512 bytes.
			// use truncate function to compress or truncate packages.
			response.Truncate(int(msgSize))

			for {

				data, err := response.Pack()
				if err != nil {
					fmt.Println("----- ERROR -----")
					fmt.Println(err)
					break
				}
				if len(data) <= int(msgSize) {
					fmt.Println("----- DEFAULT UDP MESSAGE -----")
					fmt.Println("Time:", time.Now().Format(timeFormat))
					fmt.Println("Length:", len(data))
					fmt.Println("Compressed:", response.Compress)

					if _, err = w.Write(data); err != nil {
						fmt.Println("----- ERROR -----")
						fmt.Println(err)
					}

					break
				}
				if len(response.Answer) == 0 {
					fmt.Println("Truncation is required, but not possible")
					if _, err = w.Write(data); err != nil {
						fmt.Println("----- ERROR -----")
					}
					break
				}
				fmt.Println("Truncate:", len(data))
				fmt.Println("Answer: ", len(response.Answer))
				fmt.Println("Length:", len(data))
			}
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
