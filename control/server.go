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
	c.AbortWithStatusJSON(httpResponseCode, Health{
		Healthy: healthy,
	})
}

func (s server) Status(c *gin.Context) {
	status, err := s.impl.Status()
	httpResponseCode := http.StatusOK
	if err != nil {
		httpResponseCode = http.StatusInternalServerError
	}
	c.AbortWithStatusJSON(httpResponseCode, status)
}

func (s server) PostProxies(c *gin.Context) {
	request := Proxy{}
	err := c.BindJSON(&request)
	if err != nil {
		return
	}
	response, err := s.impl.StartProxy(request.ProxyType, request.ProxyParameters)
	if err != nil {
		return
	}
	c.AbortWithStatusJSON(http.StatusOK, response)
}

func (s server) GetProxies(c *gin.Context) {
	response, err := s.impl.ListProxies()
	if err != nil {
		return
	}
	c.AbortWithStatusJSON(http.StatusOK, response)
}

func (s server) PostKeys(c *gin.Context) {
	request := SSHKey{}
	err := c.BindJSON(&request)
	if err != nil {
		return
	}
	err = s.impl.AddSSHKey(request.EncodedKey, request.PassPhrase)
	if err != nil {
		return
	}
}

func (s server) PostSSHTargets(c *gin.Context) {
	request := SSHTarget{}
	err := c.BindJSON(&request)
	if err != nil {
		return
	}
	err = s.impl.AddDialer(request.URI)
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

func (s server) GetRules(c *gin.Context) {
	response, err := s.impl.ListRules()
	if err != nil {
		return
	}
	c.AbortWithStatusJSON(http.StatusOK, response)
}

func (s server) PostRules(c *gin.Context) {
	rule := Rule{}
	err := c.BindJSON(&rule)
	if err != nil {
		return
	}
	err = s.impl.AddRule(rule)
	if err != nil {
		return
	}
}

// Start starts the API controller
func Start(impl API) {
	s := server{
		impl: impl,
	}
	r := gin.Default()
	r.GET("/health", s.Health)
	r.GET("/status", s.Status)
	r.GET("/proxies", s.GetProxies)
	r.POST("/proxies", s.PostProxies)
	r.POST("/ssh/keys", s.PostKeys)
	r.POST("/ssh/targets", s.PostSSHTargets)
	r.GET("/state", s.GetState)
	r.PUT("/state", s.PutState)
	r.POST("/targets", s.PostTargets)
	r.GET("/rules", s.GetRules)
	r.POST("/rules", s.PostRules)
	panic(r.RunUnix("/tmp/sshtunnel.sock"))
}
