package proxy

import (
	"context"
	"fmt"
	"net"
	"strconv"

	// cSpell:ignore armon
	socks5 "github.com/armon/go-socks5"
	"github.com/dueckminor/go-sshtunnel/dialer"
	"github.com/dueckminor/go-sshtunnel/rules"
)

type socks5Proxy struct {
	Dialer dialer.Dialer
	Port   int
}

func (proxy *socks5Proxy) GetPort() int {
	return proxy.Port
}

func (proxy *socks5Proxy) SetDialer(dialer dialer.Dialer) {
	proxy.Dialer = dialer
}

func init() {
	RegisterProxyFactory("socks5", newSocks5Proxy)
}

func newSocks5Proxy(parameters string) (Proxy, error) {
	proxy := &socks5Proxy{}
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

func (proxy *socks5Proxy) start(port int) (err error) {
	listener, port, err := createTCPListener(port)
	if err != nil {
		return err
	}

	proxy.Port = port

	config := &socks5.Config{}
	config.Rewriter = proxy
	config.Dial = func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
		return proxy.Dialer.Dial(network, addr)
	}
	if len(dnsTarget) > 0 {
		config.Resolver = proxy
	}

	socksServer, err := socks5.New(config)
	if err != nil {
		listener.Close()
		return err
	}

	go func() {
		defer listener.Close()
		socksServer.Serve(listener)
	}()
	return nil
}

// implements the socks5 NameResolver interface
func (proxy *socks5Proxy) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	fmt.Printf("SOCKS5: resolving '%s'...\n", name)
	ip, err := ResolveDNS(ctx, name)
	if err != nil {
		fmt.Printf("SOCKS5: resolving '%s' failed: %v\n", name, err)
	} else {
		fmt.Printf("SOCKS5: resolving '%s' -> %v\n", name, ip.String())
	}
	return ctx, ip, err
}

func (proxy *socks5Proxy) Rewrite(ctx context.Context, request *socks5.Request) (context.Context, *socks5.AddrSpec) {
	fmt.Printf("SOCKS5: connect to '%v'...\n", request.DestAddr.String())
	return ctx, request.DestAddr
}
