package control

type API interface {
	Health() (bool, error)
	Status() (Status, error)
	Stop() error
	//// SSH Keys ////
	AddSSHKey(encodedKey string, passPhrase string) error
	//// Proxies ////
	StartProxy(proxyType string, proxyParameter string) (Proxy, error)
	ListProxies() ([]Proxy, error)
	//// Dialer ////
	AddDialer(uri string) error
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
	EncodedKey string `json:"encodedKey"`
	PassPhrase string `json:"passPhrase"`
}

// SSHTarget is the transport format of the POST /ssh/targets endpoint
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
