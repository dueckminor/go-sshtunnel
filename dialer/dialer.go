package dialer

import "net"

// Dialer is a generic interface which is used to establish a net.Conn
type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}

var (
	dialers = make(map[string]Dialer)
)

func Dial(dialerName, network, addr string) (net.Conn, error) {
	if dialer, ok := dialers[dialerName]; ok {
		return dialer.Dial(network, addr)
	}
	return nil, nil
}
