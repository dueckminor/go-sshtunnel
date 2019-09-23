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

	healthMessage := HealthMessage{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	json.Unmarshal(body, &healthMessage)

	fmt.Println(healthMessage)
	return healthMessage.Healthy, err
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

func (c clientAPI) AddSSHKey(encodedKey string, passPhrase string) error {
	return c.PostJSON("/keys", AddSSHKeyMessage{
		EncodedKey: encodedKey,
		PassPhrase: passPhrase,
	}, nil)
}

func (c clientAPI) AddTarget(cidr string, tunnel string) error {
	return nil
}

func (c clientAPI) GetConfigScript() (string, error) {
	return "", nil
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

func (c clientAPI) PostJSON(path string, requestBody interface{}, responseBody *interface{}) error {
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
