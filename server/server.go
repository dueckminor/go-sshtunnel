package server

import (
	"os"

	"github.com/dueckminor/go-sshtunnel/control"
	"github.com/dueckminor/go-sshtunnel/dialer"
	"github.com/dueckminor/go-sshtunnel/proxy"
	"github.com/dueckminor/go-sshtunnel/rules"
)

// Server is the central object of sshtunnel
type Server struct {
	done    chan int
	proxies []control.Proxy
}

// Initialize initializes the Server
func (server *Server) Initialize() {
	server.done = make(chan int)
}

// Health implements control.API.Health
func (server *Server) Health() (bool, error) {
	return true, nil
}

// Status implements control.API.Status
func (server *Server) Status() (status control.Status, err error) {
	status.Healthy = true
	status.Proxies = server.proxies
	return status, nil
}

// Stop implements control.API.Stop
func (server *Server) Stop() error {
	server.done <- 0
	return nil
}

// StartProxy implements control.API.StartProxy
func (server *Server) StartProxy(proxyType string, proxyParameter string) (proxyInfo control.Proxy, err error) {
	proxy, err := proxy.NewProxy(proxyType, proxyParameter)
	if err == nil {
		proxyInfo.ProxyType = proxyType
		proxyInfo.ProxyPort = proxy.GetPort()
		proxyInfo.ProxyParameters = proxyParameter
		server.proxies = append(server.proxies, proxyInfo)
	}
	return proxyInfo, err
}

// ListProxies implements control.API.ListProxies
func (server *Server) ListProxies() ([]control.Proxy, error) {
	return server.proxies, nil
}

// AddSSHKey implements control.API.AddSSHKey
func (server *Server) AddSSHKey(encodedKey string, passPhrase string) error {
	return dialer.AddSSHKey(encodedKey, passPhrase)
}

// AddDialer implements control.API.AddDialer
func (server *Server) AddDialer(uri string) error {
	return dialer.AddDialer("default", uri)
}

// ListRules implements control.API.ListRules
func (server *Server) ListRules() ([]control.Rule, error) {
	ruleList, err := rules.GetDefaultRuleSet().ListRules()
	if err != nil {
		return nil, err
	}

	result := make([]control.Rule, len(ruleList), len(ruleList))
	for i, rule := range ruleList {
		result[i] = rules.Marshall(rule)
	}

	return result, nil
}

// AddRule implements control.API.AddRule
func (server *Server) AddRule(rule control.Rule) error {
	r, err := rules.UnMarshall(rule)
	if err != nil {
		return err
	}
	return rules.GetDefaultRuleSet().AddRule(r)
}

// Run starts the Server and waits until the Server stops
func Run(parameters []string) {
	logfile := ""
	if len(parameters) == 2 && parameters[0] == "--logfile" {
		logfile = parameters[1]
		f, err := os.Create(logfile)
		if err != nil {
			panic(err)
		}
		os.Stderr = f
		os.Stdout = f
	}

	savePID(pidFile, os.Getpid())
	defer os.Remove(pidFile)

	server := &Server{}
	server.Initialize()

	go control.Start(server)

	rc := <-server.done

	os.Exit(rc)
}
