package dialer

import (
	"net"
	"strings"

	"github.com/dueckminor/go-sshtunnel/control"
)

// Dialer is a generic interface which is used to establish a net.Conn
type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}

// DialerInfo is the internal representation of a dialer
type DialerInfo struct {
	control.Dialer
	impl Dialer
}

var (
	dialers   = make(map[string]DialerInfo)
	sshDialer *SSHDialer
)

// Dial uses the selected dialer to establish a network connection
func Dial(dialerName, network, addr string) (net.Conn, error) {
	if dialer, ok := dialers[dialerName]; ok {
		return dialer.impl.Dial(network, addr)
	}
	return nil, nil
}

func AddSSHKey(encodedKey string, passPhrase string) error {
	return makeSSHDialer().AddSSHKey(encodedKey, passPhrase)
}

func makeSSHDialer() *SSHDialer {
	if sshDialer == nil {
		sshDialer, _ = NewSSHDialer(5)
		info := DialerInfo{impl: sshDialer}
		info.Name = "default"
		info.Type = "ssh"
		info.Destination = "..."
		if _, ok := dialers["default"]; ok {
			dialers["ssh"] = info
		} else {
			dialers["default"] = info
		}
	}
	return sshDialer
}

func AddDialer(dialerName, uri string) (err error) {
	if len(dialerName) == 0 {
		dialerName = "default"
	}

	if strings.HasPrefix(uri, "socks5://") {
		dialer, err := NewSocks5Dialer(uri[9:])
		if err != nil {
			return err
		}
		info := DialerInfo{impl: dialer}
		info.Name = dialerName
		info.Type = "socks5"
		info.Destination = uri
		dialers[dialerName] = info
		return nil
	}

	makeSSHDialer()
	for _, u := range strings.Split(uri, ",") {
		sshDialer.AddDialer(u)
	}

	info := DialerInfo{impl: sshDialer}
	info.Name = dialerName
	info.Type = "ssh"
	info.Destination = uri
	dialers[dialerName] = info
	return nil
}

func ListDialers() (dialerInfos []DialerInfo, err error) {
	dialerInfos = make([]DialerInfo, len(dialers), len(dialers))
	i := 0
	for _, dialerInfo := range dialers {
		dialerInfos[i] = dialerInfo
		i++
	}
	return dialerInfos, nil
}

func Marshall(info DialerInfo) (result control.Dialer, err error) {
	result.Name = info.Name
	result.Destination = info.Destination
	result.Type = info.Type
	return result, nil
}
