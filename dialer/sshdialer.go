package dialer

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ScaleFT/sshkeys"
	"github.com/dueckminor/go-sshtunnel/logger"
	"golang.org/x/crypto/ssh"
)

type SSHAddress struct {
	user string
	host string
}

type SSHDialer struct {
	addresses []SSHAddress // ip:port
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

	//dialers["default"] = sshDialer

	sshDialer.config.Timeout = time.Duration(timeout) * time.Second
	sshDialer.config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	return sshDialer, nil
}

func passPhraseToBuffer(passPhrase string) []byte {
	if passPhrase == "" {
		return nil
	} else {
		return []byte(passPhrase)
	}
}

// CheckSSHKey/ verifies that the encodedKey can be decoded and converts it
// to a format that ssh.ParsePrivateKeyWithPassphrase can parse
func CheckSSHKey(encodedKey string, passPhrase string) error {
	_, err := sshkeys.ParseEncryptedPrivateKey([]byte(encodedKey), passPhraseToBuffer(passPhrase))
	return err
}

func (sshDialer *SSHDialer) AddSSHKey(encodedKey string, passPhrase string) error {
	signer, err := sshkeys.ParseEncryptedPrivateKey([]byte(encodedKey), passPhraseToBuffer(passPhrase))
	if err != nil {
		log.Printf("ParsePrivateKey failed:%s\n", err)
		return err
	}
	fmt.Println(signer)
	sshDialer.config.Auth = append(sshDialer.config.Auth, ssh.PublicKeys(signer))
	return nil
}

func (sshDialer *SSHDialer) AddDialer(uri string) error {
	logger.L.Printf("uri: %s\n", uri)
	if !strings.Contains(uri, "://") {
		uri = "ssh://" + uri
	}

	sshURL, err := url.Parse(uri)
	if err != nil || sshURL.Scheme != "ssh" {
		return fmt.Errorf("'%s' is not a valid ssh url", uri)
	}

	address := SSHAddress{}
	address.user = sshURL.User.Username()
	address.host = sshURL.Host
	if sshURL.Port() == "" {
		address.host += ":22"
	}

	if len(sshDialer.config.User) == 0 {
		sshDialer.config.User = address.user
	}

	logger.L.Printf("address.user: %s\n", address.user)
	logger.L.Printf("address.host: %s\n", address.host)

	sshDialer.addresses = append(sshDialer.addresses, address)

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
		log.Printf("dial %s failed: %s, reconnecting ssh server %v...\n", addr, err, sshDialer.addresses)

		if _, ok := err.(*ssh.OpenChannelError); ok {
			// we this kind of error, if the sshtunnel is up and running,
			// but no connection to the destination addr could be established
			// -> no need to dial again
			return nil, err
		}
	}

	// we are currently not connected to the ssh server
	// -> dial again
	var sshHost string

	returnValue, err := sshDialer.syncCalls.CallSynchronized(network+addr,
		func() (interface{}, error) {
			// The following code does the same as:
			//   return ssh.Dial("tcp", sshDialer.address, sshDialer.config)
			// but allows to use a comma seperated list of hostnames
			var conn net.Conn
			var err error

			cfg := new(ssh.ClientConfig)
			*cfg = *sshDialer.config

			for _, addr := range sshDialer.addresses {
				if len(addr.user) > 0 {
					cfg.User = addr.user
				}

				sshHost = addr.host

				logger.L.Printf("Trying to connect to %s@%s\n", cfg.User, sshHost)

				conn, err = net.DialTimeout("tcp", addr.host, sshDialer.config.Timeout)
				if err != nil {
					logger.L.Printf("Connect to %s@%s failed. Reason: %v\n", cfg.User, sshHost, err)
					continue
				}
				// cSpell: ignore chans,reqs
				var c ssh.Conn
				var chans <-chan ssh.NewChannel
				var reqs <-chan *ssh.Request
				c, chans, reqs, err = ssh.NewClientConn(conn, sshHost, cfg)
				if err != nil {
					logger.L.Printf("Handshake with %s@%s failed. Reason: %v\n", cfg.User, sshHost, err)
					continue
				}
				logger.L.Printf("Handshake with %s@%s succeeded\n", cfg.User, sshHost)
				return ssh.NewClient(c, chans, reqs), nil
			}
			return nil, err
		})

	if err != nil {
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
