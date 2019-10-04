package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
)

type clientAPI struct {
	httpClient http.Client
}

func (c clientAPI) Health() (healthy bool, err error) {
	resp, err := c.Get("/health")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	healthMessage := Health{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	json.Unmarshal(body, &healthMessage)

	fmt.Println(healthMessage)
	return healthMessage.Healthy, err
}

func (c clientAPI) Status() (status Status, err error) {
	err = c.GetJSON("/status", &status)
	if err != nil {
		return Status{}, err
	}
	return status, nil
}

func (c clientAPI) Stop() (err error) {
	req, err := http.NewRequest("PUT", c.MakeURL("/state"), nil)
	if err != nil {
		fmt.Println(err)
	}
	_, err = c.httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func (c clientAPI) StartProxy(proxyType string, proxyParameter string) (proxyInfo Proxy, err error) {
	err = c.PostJSON("/proxies", Proxy{
		ProxyType:       proxyType,
		ProxyParameters: proxyParameter,
	}, &proxyInfo)
	return proxyInfo, err
}

func (c clientAPI) ListProxies() (proxyInfos []Proxy, err error) {
	err = c.GetJSON("/proxies", &proxyInfos)
	return proxyInfos, err
}

func (c clientAPI) AddSSHKey(encodedKey string, passPhrase string) error {
	return c.PostJSON("/ssh/keys", SSHKey{
		EncodedKey: encodedKey,
		PassPhrase: passPhrase,
	}, nil)
}

func (c clientAPI) AddDialer(uri string) error {
	return c.PostJSON("/ssh/targets", SSHTarget{
		URI: uri,
	}, nil)
}

func (c clientAPI) ListRules() (rules []Rule, err error) {
	err = c.GetJSON("/rules", &rules)
	return rules, err
}

func (c clientAPI) AddRule(rule Rule) error {
	err := c.PostJSON("/rules", rule, nil)
	return err
}

func (c clientAPI) MakeURL(path string) (url string) {
	if len(path) > 0 && path[0] == '/' {
		return "http://unix" + path
	}
	return "http://unix/" + path
}

func (c clientAPI) Get(path string) (resp *http.Response, err error) {
	return c.httpClient.Get(c.MakeURL(path))
}

func (c clientAPI) GetJSON(path string, responseBody interface{}) error {
	req, err := http.NewRequest("GET", c.MakeURL(path), nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return json.Unmarshal(body, responseBody)
}

func (c clientAPI) PostJSON(path string, requestBody interface{}, responseBody interface{}) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.MakeURL(path), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)

	return json.Unmarshal(body, responseBody)
}

func Client() API {
	result := clientAPI{
		httpClient: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", "/tmp/sshtunnel.sock")
				},
			},
		},
	}
	return result
}
