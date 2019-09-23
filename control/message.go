package control

// HealthMessage is the transport format of the GET /health endpoint
type HealthMessage struct {
	Healthy bool `json:"healthy"`
}

// AddSSHKeyMessage is the transport format of the POST /keys endpoint
type AddSSHKeyMessage struct {
	EncodedKey string `json:"encodedKey"`
	PassPhrase string `json:"passPhrase"`
}
