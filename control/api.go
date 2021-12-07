package control

// API is the intrace of the REST API
type API interface {
	Health() (bool, error)
	Status() (Status, error)
	Stop() error
	//// SSH Keys ////
	AddSSHKey(encodedKey string, passPhrase string) error
	ListKeys() ([]SSHKey, error)
	//// Proxies ////
	StartProxy(proxyType string, proxyParameter string) (Proxy, error)
	ListProxies() ([]Proxy, error)
	//// Dialer ////
	AddDialer(uri string) error
	ListDialers() ([]Dialer, error)
	Connect(in ConnectIn) (out ConnectOut, err error)

	//// Rules ////
	ListRules() ([]Rule, error)
	AddRule(rule Rule) error
}

// Health is the transport format of the GET /health endpoint
type Health struct {
	Healthy bool `json:"healthy"`
}

// Status is the transport format of the GET /status endpoint
type Status struct {
	Health
	Proxies []Proxy `json:"proxies"`
}

// SSHKey is the transport format of the POST /ssh/keys endpoint
type SSHKey struct {
	Type       string `json:"type"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// SSHTarget is the transport format of the POST /dialers endpoint
type SSHTarget struct {
	URI string `json:"uri"`
}

// Proxy is the transport format of the POST /proxies endpoint
type Proxy struct {
	ProxyType       string `json:"type"`
	ProxyPort       int    `json:"port"`
	ProxyParameters string `json:"params"`
}

// Rule defines which IP Addresses get forwarded to a dialer
type Rule struct {
	CIDR   string `json:"cidr"`
	Dialer string `json:"dialer"`
}

// Dialer defines a dialer
type Dialer struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Destination string `json:"destination"`
}

type ConnectStatus string

const (
	ConnectStatusConnecting     ConnectStatus = "connecting"
	ConnectStatusHandshake      ConnectStatus = "handshake"
	ConnectStatusNeedPassphrase ConnectStatus = "need_passphrase"
	ConnectStatusSucceeded      ConnectStatus = "succeeded"
	ConnectStatusFailed         ConnectStatus = "failed"
)

// ConnectIn defines the input parameters of the Connect API call
type ConnectIn struct {
	ID         string `json:"id"`
	Passphrase string `json:"passphrase"`
}

type ConnectOut struct {
	ID       string        `json:"id"`
	Status   ConnectStatus `json:"status"`
	Messages []string      `json:"messages"`
}
