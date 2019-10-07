package dialer

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/scaleft/sshkeys"
	"golang.org/x/crypto/ssh"
)

type SSHDialer struct {
	address   string // ip:port
	config    *ssh.ClientConfig
	client    *ssh.Client
	syncCalls CallGroup
	lock      sync.RWMutex
}

func NewSSHDialer(timeout int) (sshDialer *SSHDialer, err error) {
	sshDialer = &SSHDialer{
		config:    &ssh.ClientConfig{},
		client:    nil,
		syncCalls: CallGroup{},
		lock:      sync.RWMutex{}}

	dialers["default"] = sshDialer

	sshDialer.config.Timeout = time.Duration(timeout) * time.Second
	sshDialer.config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	return sshDialer, nil
}

// ConvertSSHKey verifies that the encodedKey can be decoded and converts it
// to a format that ssh.ParsePrivateKeyWithPassphrase can parse
func CheckSSHKey(encodedKey string, passPhrase string) error {
	_, err := sshkeys.ParseEncryptedPrivateKey([]byte(encodedKey), []byte(passPhrase))
	return err
}

func (sshDialer *SSHDialer) AddSSHKey(encodedKey string, passPhrase string) error {
	signer, err := sshkeys.ParseEncryptedPrivateKey([]byte(encodedKey), []byte(passPhrase))
	if err != nil {
		log.Printf("ParsePrivateKey failed:%s\n", err)
		return err
	}
	fmt.Println(signer)
	sshDialer.config.Auth = append(sshDialer.config.Auth, ssh.PublicKeys(signer))
	return nil
}

func (sshDialer *SSHDialer) AddDialer(uri string) error {
	sshURL, err := url.Parse(uri)
	if err != nil || sshURL.Scheme != "ssh" {
		return fmt.Errorf("'%s' is not a valid ssh url", uri)
	}
	if sshURL.Port() == "" {
		sshDialer.address = sshURL.Host + ":22"
	} else {
		sshDialer.address = sshURL.Host
	}

	sshDialer.config.User = sshURL.User.Username()

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
			// The following code does the same as:
			//   return ssh.Dial("tcp", sshDialer.address, sshDialer.config)
			// but allows to use a comma seperated list of hostnames
			var conn net.Conn
			var err error

			sshDialerAddressParts := strings.Split(sshDialer.address, ":")
			sshDialerPort := "22"
			sshDialerAddress := sshDialerAddressParts[0]
			if len(sshDialerAddressParts) > 1 {
				sshDialerPort = sshDialerAddressParts[1]
			}

			for _, sshDialerHost := range strings.Split(sshDialerAddressParts[0], ",") {
				sshDialerAddress = sshDialerHost + ":" + sshDialerPort
				conn, err = net.DialTimeout("tcp", sshDialerAddress, sshDialer.config.Timeout)
				if err == nil {
					break
				}
			}
			if err != nil {
				return nil, err
			}
			// cSpell: ignore chans,reqs
			c, chans, reqs, err := ssh.NewClientConn(conn, sshDialerAddress, sshDialer.config)
			if err != nil {
				return nil, err
			}
			return ssh.NewClient(c, chans, reqs), nil
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

// CallSynchronized executes and returns the results of the given function, making
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
