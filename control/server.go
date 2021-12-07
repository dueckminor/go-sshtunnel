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

func (s server) GetKeys(c *gin.Context) {
	response, err := s.impl.ListKeys()
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
	err = s.impl.AddSSHKey(request.PrivateKey, request.Passphrase)
	if err != nil {
		return
	}
}

func (s server) PostDialers(c *gin.Context) {
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

func (s server) GetDialers(c *gin.Context) {
	response, err := s.impl.ListDialers()
	if err != nil {
		return
	}
	c.AbortWithStatusJSON(http.StatusOK, response)
}

func (s server) Connect(c *gin.Context) {
	in := ConnectIn{}
	err := c.BindJSON(&in)
	if err != nil {
		return
	}
	out, err := s.impl.Connect(in)
	if err != nil {
		return
	}
	c.AbortWithStatusJSON(http.StatusOK, out)
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

func (s server) createGinEngine() *gin.Engine {
	r := gin.Default()
	r.GET("/api/health", s.Health)
	r.GET("/api/status", s.Status)
	r.GET("/api/proxies", s.GetProxies)
	r.POST("/api/proxies", s.PostProxies)
	r.GET("/api/ssh/keys", s.GetKeys)
	r.POST("/api/ssh/keys", s.PostKeys)
	r.POST("/api/ssh/connect", s.Connect)
	r.POST("/api/dialers", s.PostDialers)
	r.GET("/api/dialers", s.GetDialers)
	r.GET("/api/state", s.GetState)
	r.PUT("/api/state", s.PutState)
	r.POST("/api/targets", s.PostTargets)
	r.GET("/api/rules", s.GetRules)
	r.POST("/api/rules", s.PostRules)
	return r
}

// Start starts the API controller
func Start(impl API) {
	s := server{
		impl: impl,
	}

	r := s.createGinEngine()

	panic(r.RunUnix("/tmp/sshtunnel.sock"))
}
