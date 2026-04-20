package dialer

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ScaleFT/sshkeys"
	"github.com/dueckminor/go-sshtunnel/control"
	"github.com/dueckminor/go-sshtunnel/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

func quote(input string) string {
	input = strings.Replace(input, "\n", "", -1)
	input = strings.Replace(input, "\r", "", -1)
	return strconv.Quote(input)
}

type SSHAddress struct {
	user string
	host string
}

type SSHDialer struct {
	addresses []SSHAddress // ip:port
	config    *ssh.ClientConfig
	client    *ssh.Client
	signers   []ssh.Signer
	lock      sync.RWMutex

	sshConnector *SSHConnector
}

type SSHConnector struct {
	messages    []string
	passphrase  string
	lock        sync.RWMutex
	interactive bool
	status      control.ConnectStatus
	err         error
	config      *ssh.ClientConfig
	sshDialer   *SSHDialer
	waiting     []chan bool
	// pendingHostKey holds context for an unknown-host-key prompt.
	pendingHostKey *pendingHostKeyInfo
}

// pendingHostKeyInfo holds the data needed to resolve an unknown-host-key prompt.
type pendingHostKeyInfo struct {
	hostname    string
	remote      net.Addr
	key         ssh.PublicKey
	fingerprint string
	decision    chan bool // receives true=accept, false=reject
}

func NewSSHDialer(timeout int) (sshDialer *SSHDialer, err error) {
	dirName, _ := os.UserHomeDir()
	knownHostsFile := filepath.Join(dirName, ".ssh", "known_hosts")

	hostKeyCallback, err := knownhosts.New(knownHostsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts from %s: %w", knownHostsFile, err)
	}

	sshDialer = &SSHDialer{
		config: &ssh.ClientConfig{
			HostKeyCallback: hostKeyCallback,
			Timeout:         time.Duration(timeout) * time.Second,
		},
		client: nil,
		lock:   sync.RWMutex{},
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

// CheckSSHKey verifies that the encodedKey can be decoded and converts it
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

	sshDialer.signers = append(sshDialer.signers, signer)
	sshDialer.config.Auth = append(sshDialer.config.Auth, ssh.PublicKeys(signer))
	return nil
}

func (sshDialer *SSHDialer) GetSSHKeys() (keys []control.SSHKey, err error) {
	for _, signer := range sshDialer.signers {
		pub := signer.PublicKey()

		sshkey := control.SSHKey{}
		sshkey.Type = pub.Type()
		sshkey.PublicKey = base64.StdEncoding.EncodeToString((pub.Marshal()))

		keys = append(keys, sshkey)
	}

	return keys, nil
}

func (sshDialer *SSHDialer) AddDialer(uri string) error {
	logger.L.Printf("uri: %s\n", quote(uri))
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

	logger.L.Printf("address.user: %s\n", quote(address.user))
	logger.L.Printf("address.host: %s\n", quote(address.host))

	sshDialer.addresses = append(sshDialer.addresses, address)

	return nil
}

func (sshDialer *SSHDialer) Dial(network, addr string) (net.Conn, error) {
	sshDialer.lock.RLock()
	client := sshDialer.client
	sshDialer.lock.RUnlock()

	if nil != client {
		c, err := client.Dial(network, addr)
		if err == nil {
			return c, nil
		}
		// reconnect if required
		log.Printf("dial %s failed: %s, reconnecting ssh server %v...\n", quote(addr), err, sshDialer.addresses)

		if _, ok := err.(*ssh.OpenChannelError); ok {
			// we this kind of error, if the sshtunnel is up and running,
			// but no connection to the destination addr could be established
			// -> no need to dial again
			return nil, err
		}

		sshDialer.lock.Lock()
		sshDialer.client = nil
		sshDialer.lock.Unlock()
	}

	client, err := sshDialer.Connect()
	if err != nil {
		return nil, err
	}

	return client.Dial(network, addr)
}

func (sshDialer *SSHDialer) Connect() (*ssh.Client, error) {
	sshDialer.lock.RLock()
	if nil != sshDialer.client {
		return sshDialer.client, nil
	}
	sshDialer.lock.RUnlock()

	sshConnector := sshDialer.GetConnector(false)

	for !sshConnector.Done() {
		sshConnector.Wait()
	}

	if sshDialer.client != nil {
		return sshDialer.client, nil
	}
	return nil, sshConnector.err
}

func (sshDialer *SSHDialer) GetConnector(interactive bool) *SSHConnector {
	sshDialer.lock.RLock()
	if nil != sshDialer.sshConnector {
		if interactive {
			sshDialer.sshConnector.interactive = true
		}
		return sshDialer.sshConnector
	}
	sshDialer.lock.RUnlock()

	sshDialer.lock.Lock()
	defer sshDialer.lock.Unlock()

	sshDialer.sshConnector = &SSHConnector{
		interactive: interactive,
		sshDialer:   sshDialer,
		lock:        sync.RWMutex{},
	}

	go sshDialer.sshConnector.connect()

	return sshDialer.sshConnector
}

func (sshConnector *SSHConnector) Status() control.ConnectStatus {
	return sshConnector.status
}

func (sshConnector *SSHConnector) MessageCount() int {
	sshConnector.lock.RLock()
	defer sshConnector.lock.RUnlock()
	return len(sshConnector.messages)
}

func (sshConnector *SSHConnector) Message(i int) string {
	sshConnector.lock.RLock()
	defer sshConnector.lock.RUnlock()
	return sshConnector.messages[i]
}

func (sshConnector *SSHConnector) SetPassphrase(passphrase string) error {
	sshConnector.lock.Lock()
	defer sshConnector.lock.Unlock()
	if sshConnector.status != control.ConnectStatusNeedPassphrase {
		return fmt.Errorf("wrong status. Expected %s, Have %s", control.ConnectStatusNeedPassphrase, sshConnector.status)
	}
	sshConnector.passphrase = passphrase
	sshConnector.status = control.ConnectStatusHandshake
	sshConnector.notifyWaitingLocked()
	return nil
}

func (sshConnector *SSHConnector) Done() bool {
	return sshConnector.status == control.ConnectStatusSucceeded ||
		sshConnector.status == control.ConnectStatusFailed
}

func (sshConnector *SSHConnector) HostKeyFingerprint() string {
	sshConnector.lock.RLock()
	defer sshConnector.lock.RUnlock()
	if sshConnector.pendingHostKey == nil {
		return ""
	}
	return sshConnector.pendingHostKey.fingerprint
}

func (sshConnector *SSHConnector) AcceptHostKey(accept bool) error {
	sshConnector.lock.Lock()
	defer sshConnector.lock.Unlock()
	if sshConnector.status != control.ConnectStatusUnknownHostKey {
		return fmt.Errorf("wrong status. Expected %s, Have %s",
			control.ConnectStatusUnknownHostKey, sshConnector.status)
	}
	sshConnector.status = control.ConnectStatusHandshake
	sshConnector.pendingHostKey.decision <- accept
	sshConnector.notifyWaitingLocked()
	return nil
}

func (sshConnector *SSHConnector) Print(msg string) {
	sshConnector.lock.Lock()
	defer sshConnector.lock.Unlock()
	sshConnector.messages = append(sshConnector.messages, msg)
	sshConnector.notifyWaitingLocked()
}

func (sshConnector *SSHConnector) Printf(format string, args ...interface{}) {
	sshConnector.Print(fmt.Sprintf(format, args...))
}

func (sshConnector *SSHConnector) connect() {
	var sshHost string
	defer func() {
		if sshConnector.status != control.ConnectStatusSucceeded {
			sshConnector.status = control.ConnectStatusFailed
		}
		sshConnector.sshDialer.sshConnector = nil
		sshConnector.notifyWaiting()
	}()

	// The following code does the same as:
	//   return ssh.Dial("tcp", sshDialer.address, sshDialer.config)
	// but allows to use a comma seperated list of hostnames
	var conn net.Conn
	var err error

	cfg := new(ssh.ClientConfig)
	*cfg = *sshConnector.sshDialer.config

	// Wrap the base HostKeyCallback to handle the interactive unknown-host-key
	// prompt (analogous to the "The authenticity of host … can't be established"
	// message in the OpenSSH client). A key mismatch (possible MitM) is always
	// rejected, even in interactive mode.
	baseCallback := cfg.HostKeyCallback
	cfg.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := baseCallback(hostname, remote, key)
		if err == nil {
			return nil // key is known and matches
		}

		var keyErr *knownhosts.KeyError
		if !isKeyError(err, &keyErr) {
			return err // some other error
		}

		if len(keyErr.Want) > 0 {
			// Key mismatch – a different key was expected. This is a potential
			// MitM attack; always abort regardless of interactive mode.
			sshConnector.Printf(
				"WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED for %s!\n"+
					"Someone could be eavesdropping on you right now (man-in-the-middle attack)!\n"+
					"Expected fingerprint: %s\n"+
					"Got fingerprint:      %s\n",
				hostname,
				ssh.FingerprintSHA256(keyErr.Want[0].Key),
				ssh.FingerprintSHA256(key),
			)
			return err
		}

		// Host is not in known_hosts at all.
		fingerprint := ssh.FingerprintSHA256(key)

		if !sshConnector.interactive {
			// Non-interactive: fail with a helpful message.
			sshConnector.Printf(
				"Host %s is not in known_hosts (fingerprint: %s).\n"+
					"Add it manually or use an interactive session to confirm.\n",
				hostname, fingerprint,
			)
			return err
		}

		// Interactive: pause and ask the user.
		decision := make(chan bool, 1)
		sshConnector.lock.Lock()
		sshConnector.pendingHostKey = &pendingHostKeyInfo{
			hostname:    hostname,
			remote:      remote,
			key:         key,
			fingerprint: fingerprint,
			decision:    decision,
		}
		sshConnector.status = control.ConnectStatusUnknownHostKey
		sshConnector.messages = append(sshConnector.messages,
			fmt.Sprintf(
				"The authenticity of host %s can't be established.\n"+
					"Host key fingerprint (SHA256): %s",
				hostname, fingerprint,
			),
		)
		sshConnector.notifyWaitingLocked()
		sshConnector.lock.Unlock()

		accepted := <-decision

		sshConnector.lock.Lock()
		sshConnector.pendingHostKey = nil
		sshConnector.lock.Unlock()

		if !accepted {
			return fmt.Errorf("host key for %s rejected by user", hostname)
		}

		// User accepted – persist the key to known_hosts.
		if writeErr := appendKnownHost(hostname, key); writeErr != nil {
			log.Printf("Warning: could not write host key to known_hosts: %v\n", writeErr)
		} else {
			sshConnector.Printf("Permanently added %s to known_hosts.\n", hostname)
		}
		return nil
	}

	cfg.Auth = append(sshConnector.sshDialer.config.Auth, ssh.PasswordCallback(func() (secret string, err error) {
		sshConnector.status = control.ConnectStatusNeedPassphrase
		for !sshConnector.Done() {
			if len(sshConnector.passphrase) > 0 {
				passphrase := sshConnector.passphrase
				sshConnector.passphrase = ""
				sshConnector.status = control.ConnectStatusHandshake
				sshConnector.notifyWaiting()
				return passphrase, nil
			}
			sshConnector.Wait()
		}
		return "", nil
	}))

	socket := os.Getenv("SSH_AUTH_SOCK")
	if len(socket) > 0 {
		fmt.Println("Trying to use SSH_AUTH_SOCK:", socket)
		conn, err := net.Dial("unix", socket)
		if err == nil {
			fmt.Println("connected to SSH_AUTH_SOCK")
			agentClient := agent.NewClient(conn)
			cfg.Auth = append(cfg.Auth, ssh.PublicKeysCallback(agentClient.Signers))
		} else {
			fmt.Println("Failed to connect to SSH_AUTH_SOCK:", err)
		}
	}

	cfg.BannerCallback = func(message string) error {
		sshConnector.Print(message)
		return nil
	}

	for _, addr := range sshDialer.addresses {
		if len(addr.user) > 0 {
			cfg.User = addr.user
		}

		sshHost = addr.host

		sshConnector.Printf("Trying to connect to %s@%s\n", cfg.User, sshHost)
		sshConnector.status = control.ConnectStatusConnecting

		conn, err = net.DialTimeout("tcp", addr.host, sshDialer.config.Timeout)
		if err != nil {
			sshConnector.Printf("Connect to %s@%s failed. Reason: %v\n", cfg.User, sshHost, err)
			continue
		}

		sshConnector.status = control.ConnectStatusHandshake

		var c ssh.Conn
		var chans <-chan ssh.NewChannel
		var reqs <-chan *ssh.Request
		c, chans, reqs, err = ssh.NewClientConn(conn, sshHost, cfg)

		if err != nil {
			sshConnector.Printf("Handshake with %s@%s failed. Reason: %v\n", cfg.User, sshHost, err)
			continue
		}
		sshConnector.Printf("Handshake with %s@%s succeeded\n", cfg.User, sshHost)
		sshConnector.sshDialer.client = ssh.NewClient(c, chans, reqs)
		sshConnector.status = control.ConnectStatusSucceeded
		sshConnector.err = nil
		return
	}
}

func (sshConnector *SSHConnector) notifyWaiting() {
	sshConnector.lock.Lock()
	defer sshConnector.lock.Unlock()
	sshConnector.notifyWaitingLocked()
}

func (sshConnector *SSHConnector) notifyWaitingLocked() {
	for _, w := range sshConnector.waiting {
		w <- true
	}
	sshConnector.waiting = nil
}

func (sshConnector *SSHConnector) Wait() error {
	if sshConnector.Done() {
		return sshConnector.err
	}

	w := make(chan bool)

	sshConnector.lock.Lock()
	sshConnector.waiting = append(sshConnector.waiting, w)
	sshConnector.lock.Unlock()

	<-w
	return nil
}

func isKeyError(err error, target **knownhosts.KeyError) bool {
	var ke *knownhosts.KeyError
	ok := errors.As(err, &ke)
	if ok {
		*target = ke
	}
	return ok
}

func appendKnownHost(hostname string, key ssh.PublicKey) error {
	dirName, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	knownHostsPath := filepath.Join(dirName, ".ssh", "known_hosts")
	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := knownhosts.Line([]string{hostname}, key)
	_, err = fmt.Fprintln(f, line)
	return err
}
