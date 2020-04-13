package sshforward

import (
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHForward is used to forward requests over SSH
type SSHForward struct {
	address   string // ip:port
	config    *ssh.ClientConfig
	client    *ssh.Client
	syncCalls CallGroup
	lock      sync.RWMutex
}

// NewSSHForward creates an SSHForward object
func NewSSHForward(server, port, user, privateKeyFile string, timeout int) (*SSHForward, error) {
	forward := &SSHForward{
		address:   server + ":" + port,
		config:    &ssh.ClientConfig{User: user},
		client:    nil,
		syncCalls: CallGroup{},
		lock:      sync.RWMutex{}}

	pem, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		log.Printf("ReadFile %s failed:%s\n", privateKeyFile, err)
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		log.Printf("ParsePrivateKey %s failed:%s\n", privateKeyFile, err)
		return nil, err
	}

	forward.config.Auth = append(forward.config.Auth, ssh.PublicKeys(signer))
	forward.config.Timeout = time.Duration(timeout) * time.Second
	forward.config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	forward.client, err = ssh.Dial("tcp", forward.address, forward.config)
	if err != nil {
		log.Printf("ssh connection failed:%v", err)
		return nil, err
	}

	return forward, nil
}

// Dial implements new.Dial over SSH
func (forward *SSHForward) Dial(network, addr string) (net.Conn, error) {
	forward.lock.RLock()
	cli := forward.client
	forward.lock.RUnlock()

	c, err := cli.Dial(network, addr)
	if err == nil {
		return c, nil
	}

	// reconnect if required
	log.Printf("dial %s failed: %s, reconnecting ssh server %s...\n", addr, err, forward.address)
	returnValue, err := forward.syncCalls.CallSynchronized(network+addr,
		func() (interface{}, error) {
			return ssh.Dial("tcp", forward.address, forward.config)
		})
	if err != nil {
		log.Printf("connect ssh server %s failed: %s\n", forward.address, err)
		return nil, err
	}

	cli = returnValue.(*ssh.Client)

	forward.lock.Lock()
	forward.client = cli
	forward.lock.Unlock()

	return cli.Dial(network, addr)
}

type call struct {
	waitGroup   sync.WaitGroup
	returnValue interface{}
	err         error
}

// CallGroup is used to ensure that an initialization is called exactly once
type CallGroup struct {
	mutex   sync.Mutex
	id2call map[string]*call
}

// CallSynchronized executes and returns the results of the given function,
// making sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
func (group *CallGroup) CallSynchronized(key string, callMe func() (interface{}, error)) (interface{}, error) {
	group.mutex.Lock()
	if group.id2call == nil {
		group.id2call = make(map[string]*call)
	}
	if c, ok := group.id2call[key]; ok {
		group.mutex.Unlock()
		c.waitGroup.Wait()
		return c.returnValue, c.err
	}
	c := new(call)
	c.waitGroup.Add(1)
	group.id2call[key] = c
	group.mutex.Unlock()

	c.returnValue, c.err = callMe()
	c.waitGroup.Done()

	group.mutex.Lock()
	delete(group.id2call, key)
	group.mutex.Unlock()

	return c.returnValue, c.err
}
