package proxy

import (
	"fmt"
	"net"

	"github.com/dueckminor/go-sshtunnel/dialer"
)

// Proxy is the generic interface for proxies
type Proxy interface {
	GetPort() int
	SetDialer(dialer dialer.Dialer)
}

// NewProxy creates a new proxy
func NewProxy(proxyType, proxyParameters string) (Proxy, error) {
	if factory, ok := proxyFactories[proxyType]; ok {
		return factory(proxyParameters)
	}

	return nil, fmt.Errorf("failed to create proxy with type '%s' and parameters '%s'", proxyType, proxyParameters)
}

type proxyFactory func(parameters string) (Proxy, error)

var proxyFactories = make(map[string]proxyFactory)

// RegisterProxyFactory is called by proxy implementations to make their
// implementation available
func RegisterProxyFactory(proxyType string, factory proxyFactory) {
	proxyFactories[proxyType] = factory
}

func createTCPListener(portRequested int) (listener *net.TCPListener, port int, err error) {
	address := fmt.Sprintf("127.0.0.1:%d", portRequested)

	addr, err := net.ResolveTCPAddr("tcp4", address)
	listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		return nil, 0, err
	}

	addr, err = net.ResolveTCPAddr(listener.Addr().Network(), listener.Addr().String())
	if err != nil {
		listener.Close()
		return nil, 0, err
	}

	return listener, addr.Port, nil
}
