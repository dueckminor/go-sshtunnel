package dialer

import (
	"golang.org/x/net/proxy"
)

func NewSocks5Dialer(addr string) (dialer Dialer, err error) {
	return proxy.SOCKS5("tcp", addr, nil, nil)
}
