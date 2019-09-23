package control

type API interface {
	Health() (bool, error)
	Stop() error
	AddSSHKey(encodedKey string, passPhrase string) error
	AddTarget(cidr string, tunnel string) error
	GetConfigScript() (string, error)
}
