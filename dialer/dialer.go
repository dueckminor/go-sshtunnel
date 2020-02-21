package dialer

import (
	"net"
	"strings"
)

// Dialer is a generic interface which is used to establish a net.Conn
type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}

var (
	dialers   = make(map[string]Dialer)
	sshDialer *SSHDialer
)

// Dial uses the selected dialer to establish a network connection
func Dial(dialerName, network, addr string) (net.Conn, error) {
	if dialer, ok := dialers[dialerName]; ok {
		return dialer.Dial(network, addr)
	}
	return nil, nil
}

func AddSSHKey(encodedKey string, passPhrase string) error {
	makeSSHDialer()
	return sshDialer.AddSSHKey(encodedKey, passPhrase)
}

func makeSSHDialer() {
	if sshDialer == nil {
		sshDialer, _ = NewSSHDialer(5)
		dialers["default"] = sshDialer
	}
}

func AddDialer(dialerName, uri string) (err error) {
	if strings.HasPrefix(uri, "socks5://") {
		dialer, err := NewSocks5Dialer(uri[9:])
		if err != nil {
			return err
		}
		dialers[dialerName] = dialer
		return nil
	}

	makeSSHDialer()
	sshDialer.AddDialer(uri)
	return nil
}
