package sshdialer

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHDialer struct {
	address   string // ip:port
	config    *ssh.ClientConfig
	client    *ssh.Client
	syncCalls CallGroup
	lock      sync.RWMutex
}

func NewSSHDialer(server, port, user string, timeout int) (sshDialer *SSHDialer, err error) {
	sshDialer = &SSHDialer{
		address:   server + ":" + port,
		config:    &ssh.ClientConfig{User: user},
		client:    nil,
		syncCalls: CallGroup{},
		lock:      sync.RWMutex{}}

	sshDialer.config.Timeout = time.Duration(timeout) * time.Second
	sshDialer.config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	// sshDialer.client, err = ssh.Dial("tcp", sshDialer.address, sshDialer.config)
	// if err != nil {
	// 	log.Printf("ssh connection failed:%v", err)
	// 	return nil, err
	// }

	return sshDialer, nil
}

func (sshDialer *SSHDialer) AddSSHKey(encodedKey string, passPhrase string) error {
	signer, err := ssh.ParsePrivateKey([]byte(encodedKey))
	if err != nil {
		log.Printf("ParsePrivateKey failed:%s\n", err)
		return err
	}
	fmt.Println(signer)
	sshDialer.config.Auth = append(sshDialer.config.Auth, ssh.PublicKeys(signer))
	return nil
}

func (sshDialer *SSHDialer) Dial(network, addr string) (net.Conn, error) {
	sshDialer.lock.RLock()
	cli := sshDialer.client
	sshDialer.lock.RUnlock()

	if nil != cli {
		c, err := cli.Dial(network, addr)
		if err == nil {
			return c, nil
		}
		// reconnect if required
		log.Printf("dial %s failed: %s, reconnecting ssh server %s...\n", addr, err, sshDialer.address)
	}

	returnValue, err := sshDialer.syncCalls.CallSynchronized(network+addr,
		func() (interface{}, error) {
			return ssh.Dial("tcp", sshDialer.address, sshDialer.config)
		})
	if err != nil {
		log.Printf("connect ssh server %s failed: %s\n", sshDialer.address, err)
		return nil, err
	}

	cli = returnValue.(*ssh.Client)

	sshDialer.lock.Lock()
	sshDialer.client = cli
	sshDialer.lock.Unlock()

	return cli.Dial(network, addr)
}

type call struct {
	waitGroup   sync.WaitGroup
	returnValue interface{}
	err         error
}

type CallGroup struct {
	mutex   sync.Mutex
	id2call map[string]*call
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
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
