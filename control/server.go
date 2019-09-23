package control

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type server struct {
	impl API
}

func (s server) Health(c *gin.Context) {
	healthy, err := s.impl.Health()
	httpResponseCode := http.StatusOK
	if err != nil {
		httpResponseCode = http.StatusInternalServerError
	}
	c.AbortWithStatusJSON(httpResponseCode, HealthMessage{
		Healthy: healthy,
	})
}

func (s server) PostKeys(c *gin.Context) {
	request := AddSSHKeyMessage{}
	err := c.BindJSON(&request)
	if err != nil {
		return
	}
	err = s.impl.AddSSHKey(request.EncodedKey, request.PassPhrase)
	if err != nil {
		return
	}
}

func (s server) GetState(c *gin.Context) {
}

func (s server) PutState(c *gin.Context) {
	err := s.impl.Stop()
	fmt.Println(err)
}

func (s server) PostTargets(c *gin.Context) {
	err := s.impl.Stop()
	fmt.Println(err)
}

// Start starts the API controller
func Start(impl API) {
	s := server{
		impl: impl,
	}
	r := gin.Default()
	r.GET("/health", s.Health)
	r.POST("/keys", s.PostKeys)
	r.GET("/state", s.GetState)
	r.PUT("/state", s.PutState)
	r.POST("/targets", s.PostTargets)
	panic(r.RunUnix("/tmp/sshtunnel.sock"))
}
