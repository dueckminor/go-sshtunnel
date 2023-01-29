package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/dueckminor/go-sshtunnel/dialer"
	"github.com/dueckminor/go-sshtunnel/rules"
)

type httpProxy struct {
	Dialer dialer.Dialer
	Port   int
}

func (proxy *httpProxy) GetPort() int {
	return proxy.Port
}

func (proxy *httpProxy) SetDialer(dialer dialer.Dialer) {
	proxy.Dialer = dialer
}

func init() {
	RegisterProxyFactory("http", newHttpProxy)
}

func newHttpProxy(parameters string) (Proxy, error) {
	proxy := &httpProxy{}
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

func (proxy *httpProxy) start(port int) (err error) {
	listener, port, err := createTCPListener(port)
	if err != nil {
		return err
	}

	proxy.Port = port

	server := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "RESOLVE" {
				proxy.handleResolve(w, r)
			} else if r.Method == http.MethodConnect {
				proxy.handleTunneling(w, r)
			} else {
				proxy.handleHTTP(w, r)
			}
		}),
	}

	go func() {
		defer listener.Close()
		server.Serve(listener)
	}()

	return nil
}

func (proxy *httpProxy) handleTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := proxy.Dialer.Dial("tcp", r.Host)
	//net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	copy := func(destination io.WriteCloser, source io.ReadCloser) {
		defer destination.Close()
		defer source.Close()
		io.Copy(destination, source)
	}

	go copy(dest_conn, client_conn)
	go copy(client_conn, dest_conn)
}

func (proxy *httpProxy) handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	proxy.copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (proxy *httpProxy) handleResolve(w http.ResponseWriter, req *http.Request) {
	ip, err := ResolveDNS(context.Background(), req.Host)
	fmt.Println(ip, err)
	w.Header().Add("Host", ip.String())
	w.WriteHeader(http.StatusOK)
}

func (proxy *httpProxy) copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
