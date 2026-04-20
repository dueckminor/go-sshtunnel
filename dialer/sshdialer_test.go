package dialer

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func generateTestHostKey(t *testing.T) (ssh.PublicKey, ssh.Signer) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	return signer.PublicKey(), signer
}

func writeKnownHostsFile(t *testing.T, dir string, host string, pub ssh.PublicKey) string {
	t.Helper()
	knownHostsPath := filepath.Join(dir, "known_hosts")
	line := knownhosts.Line([]string{host}, pub)
	if err := os.WriteFile(knownHostsPath, []byte(line+"\n"), 0600); err != nil {
		t.Fatalf("failed to write known_hosts: %v", err)
	}
	return knownHostsPath
}

func TestNewSSHDialer_ValidKnownHosts(t *testing.T) {
	tmpDir := t.TempDir()

	pub, _ := generateTestHostKey(t)
	writeKnownHostsFile(t, tmpDir, "testhost.example.com", pub)

	t.Setenv("HOME", tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, ".ssh"), 0700); err != nil {
		t.Fatalf("failed to create .ssh dir: %v", err)
	}
	writeKnownHostsFile(t, filepath.Join(tmpDir, ".ssh"), "testhost.example.com", pub)

	dialer, err := NewSSHDialer(10)
	if err != nil {
		t.Fatalf("NewSSHDialer failed unexpectedly: %v", err)
	}
	if dialer == nil {
		t.Fatal("expected non-nil SSHDialer")
	}
	if dialer.config == nil {
		t.Fatal("expected non-nil config")
	}
	if dialer.config.HostKeyCallback == nil {
		t.Fatal("expected HostKeyCallback to be set")
	}
}

func TestNewSSHDialer_HostKeyCallback_AcceptsKnownHost(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("failed to create .ssh dir: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	pub, _ := generateTestHostKey(t)
	writeKnownHostsFile(t, sshDir, "knownhost.example.com:22", pub)

	dialer, err := NewSSHDialer(10)
	if err != nil {
		t.Fatalf("NewSSHDialer failed: %v", err)
	}

	addr := &testAddr{addr: "knownhost.example.com:22"}
	if err := dialer.config.HostKeyCallback("knownhost.example.com:22", addr, pub); err != nil {
		t.Errorf("HostKeyCallback rejected a known host key: %v", err)
	}
}

func TestNewSSHDialer_HostKeyCallback_RejectsUnknownHost(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("failed to create .ssh dir: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	knownPub, _ := generateTestHostKey(t)
	writeKnownHostsFile(t, sshDir, "knownhost.example.com:22", knownPub)

	dialer, err := NewSSHDialer(10)
	if err != nil {
		t.Fatalf("NewSSHDialer failed: %v", err)
	}

	unknownPub, _ := generateTestHostKey(t)
	addr := &testAddr{addr: "knownhost.example.com:22"}
	if err := dialer.config.HostKeyCallback("knownhost.example.com:22", addr, unknownPub); err == nil {
		t.Error("HostKeyCallback should have rejected an unknown host key, but accepted it")
	}
}

func TestNewSSHDialer_HostKeyCallback_RejectsUnknownHostname(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("failed to create .ssh dir: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	pub, _ := generateTestHostKey(t)
	writeKnownHostsFile(t, sshDir, "knownhost.example.com:22", pub)

	dialer, err := NewSSHDialer(10)
	if err != nil {
		t.Fatalf("NewSSHDialer failed: %v", err)
	}

	addr := &testAddr{addr: "unknownhost.example.com:22"}
	if err := dialer.config.HostKeyCallback("unknownhost.example.com:22", addr, pub); err == nil {
		t.Error("HostKeyCallback should have rejected a key for an unknown hostname, but accepted it")
	}
}

func TestNewSSHDialer_MissingKnownHostsFile(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("failed to create .ssh dir: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	_, err := NewSSHDialer(10)
	if err == nil {
		t.Error("NewSSHDialer should return an error when known_hosts file is missing")
	}
}

func TestNewSSHDialer_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("failed to create .ssh dir: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	pub, _ := generateTestHostKey(t)
	writeKnownHostsFile(t, sshDir, "host.example.com:22", pub)

	const timeoutSeconds = 42
	dialer, err := NewSSHDialer(timeoutSeconds)
	if err != nil {
		t.Fatalf("NewSSHDialer failed: %v", err)
	}

	expected := timeoutSeconds * int(1e9) // nanoseconds
	if int(dialer.config.Timeout) != expected {
		t.Errorf("expected timeout %d ns, got %d ns", expected, dialer.config.Timeout)
	}
}

type testAddr struct {
	addr string
}

func (a *testAddr) Network() string { return "tcp" }
func (a *testAddr) String() string  { return a.addr }

// Interactive unknown-host-key flow

func setupInteractiveDialer(t *testing.T) (d *SSHDialer, sshDir string, knownPub ssh.PublicKey) {
	t.Helper()
	tmpDir := t.TempDir()
	sshDir = filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	t.Setenv("HOME", tmpDir)

	knownPub, _ = generateTestHostKey(t)
	writeKnownHostsFile(t, sshDir, "knownhost.example.com:22", knownPub)

	var err error
	d, err = NewSSHDialer(10)
	if err != nil {
		t.Fatalf("NewSSHDialer: %v", err)
	}
	return d, sshDir, knownPub
}

func makeConnector(d *SSHDialer, interactive bool) *SSHConnector {
	return &SSHConnector{
		interactive: interactive,
		sshDialer:   d,
		lock:        sync.RWMutex{},
	}
}

func TestInteractiveFlow_AcceptUnknownHost(t *testing.T) {
	d, sshDir, _ := setupInteractiveDialer(t)
	connector := makeConnector(d, true)

	newPub, _ := generateTestHostKey(t)
	addr := &testAddr{addr: "newhost.example.com:22"}

	var callbackErr error
	done := make(chan struct{})

	// Run the callback in a goroutine – it will block waiting for AcceptHostKey.
	go func() {
		callbackErr = d.config.HostKeyCallback("newhost.example.com:22", addr, newPub)
		close(done)
	}()

	_ = connector // connector is used in next tests

	// For this test, call appendKnownHost + isKeyError directly.
	<-done // callback should fail because sshConnector has no interactive prompt attached

	// The direct HostKeyCallback (without a connector) should fail for unknown hosts.
	if callbackErr == nil {
		t.Error("expected error for unknown host without interactive connector")
	}

	if err := appendKnownHost("newhost.example.com:22", newPub); err != nil {
		t.Fatalf("appendKnownHost failed: %v", err)
	}

	cb, err := knownhosts.New(filepath.Join(sshDir, "known_hosts"))
	if err != nil {
		t.Fatalf("knownhosts.New after append: %v", err)
	}
	if err := cb("newhost.example.com:22", addr, newPub); err != nil {
		t.Errorf("key not found in known_hosts after appendKnownHost: %v", err)
	}
}

func TestInteractiveFlow_RejectMismatch(t *testing.T) {
	d, _, knownPub := setupInteractiveDialer(t)
	_ = knownPub

	differentPub, _ := generateTestHostKey(t)
	addr := &testAddr{addr: "knownhost.example.com:22"}

	err := d.config.HostKeyCallback("knownhost.example.com:22", addr, differentPub)
	if err == nil {
		t.Error("key mismatch (MitM scenario) should always be rejected")
	}
}

func TestIsKeyError(t *testing.T) {
	ke := &knownhosts.KeyError{}
	var target *knownhosts.KeyError
	if !isKeyError(ke, &target) {
		t.Error("isKeyError should return true for *knownhosts.KeyError")
	}
	if !errors.Is(target, ke) {
		t.Error("isKeyError should populate target")
	}

	other := fmt.Errorf("some other error")
	if isKeyError(other, &target) {
		t.Error("isKeyError should return false for non-KeyError")
	}
}
