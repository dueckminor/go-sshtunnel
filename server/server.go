package server

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dueckminor/go-sshtunnel/commands"
	"github.com/dueckminor/go-sshtunnel/control"
	"github.com/dueckminor/go-sshtunnel/dialer"
	"github.com/dueckminor/go-sshtunnel/proxy"
	"github.com/dueckminor/go-sshtunnel/rules"
)

// Server is the central object of sshtunnel
type Server struct {
	done    chan int
	proxies []control.Proxy

	connectors map[string]*ServerConnector
}

type ServerConnector struct {
	id           string
	sshConnector *dialer.SSHConnector
	messageCount int
	lastActivity time.Time
}

// Initialize initializes the Server
func (server *Server) Initialize() {
	server.done = make(chan int)
	server.connectors = make(map[string]*ServerConnector)
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

func (server *Server) ListKeys() ([]control.SSHKey, error) {
	return dialer.GetSSHKeys()
}

// AddDialer implements control.API.AddDialer
func (server *Server) AddDialer(uri string) error {
	return dialer.AddDialer("default", uri)
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (server *Server) Connect(in control.ConnectIn) (out control.ConnectOut, err error) {
	var c *ServerConnector

	if len(in.ID) > 0 {
		c = server.connectors[in.ID]
		if c == nil {
			return out, fmt.Errorf("there is no connector with id '%s'", in.ID)
		}
	} else {
		sshConnector, err := dialer.GetConnector()
		if err != nil {
			return out, err
		}

		c = &ServerConnector{
			sshConnector: sshConnector,
		}
		c.id, err = randomHex(10)
		if err != nil {
			return out, err
		}
		server.connectors[c.id] = c
	}

	if len(in.Passphrase) > 0 {
		c.sshConnector.SetPassphrase(in.Passphrase)
	}

	out.ID = c.id

	for {
		if c.messageCount < c.sshConnector.MessageCount() {
			out.Messages = append(out.Messages, c.sshConnector.Message(c.messageCount))
			c.messageCount++
		} else {
			if len(out.Messages) > 0 || c.sshConnector.Done() || c.sshConnector.Status() == control.ConnectStatusNeedPassphrase {
				break
			}
		}
	}
	out.Status = c.sshConnector.Status()
	return out, nil
}

func (server *Server) ListDialers() (dialers []control.Dialer, err error) {
	dialerList, err := dialer.ListDialers()
	if err != nil {
		return nil, err
	}

	result := make([]control.Dialer, len(dialerList), len(dialerList))
	for i, d := range dialerList {
		result[i], _ = dialer.Marshall(d)
	}

	return result, nil
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

	flags := flag.NewFlagSet("daemon", flag.ExitOnError)
	logfile := flags.String("logfile", "", "The logfile")

	flags.Parse(parameters)

	if len(*logfile) > 0 {
		f, err := os.Create(*logfile)
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

	additionalCommands := flags.Args()

outer:
	for len(additionalCommands) > 0 {
		time.Sleep(time.Millisecond * 100)
		for i, a := range additionalCommands {
			if a == "--" {
				commands.ExecuteCommand(additionalCommands[0], additionalCommands[1:i]...)
				additionalCommands = additionalCommands[i+1:]
				continue outer
			}
		}
		commands.ExecuteCommand(additionalCommands[0], additionalCommands[1:]...)
		break
	}

	rc := <-server.done

	os.Exit(rc)
}
