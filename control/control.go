package control

import (
	"net"
	"net/http"
)

// Start starts the API controller
func Start() {
	server := http.Server{}
	unixListener, err := net.Listen("unix", "/tmp/sshtunnel.sock")
	if err != nil {
		panic(err)
	}
	panic(server.Serve(unixListener))
}
