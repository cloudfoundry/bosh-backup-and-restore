package boshclient

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	boshURL      string
	boshUsername string
	boshPassword string
	httpClient   *http.Client
}

func New(boshURL, username, password string) *Client {
	return &Client{
		boshURL:      boshURL,
		boshUsername: username,
		boshPassword: password,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

func (c *Client) CheckDeploymentExists(name string) (bool, error) {
	req, err := http.NewRequest("GET", c.boshURL+"/deployments/"+name, nil)
	if err != nil {
		return false, err
	}

	req.SetBasicAuth(c.boshUsername, c.boshPassword)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusUnauthorized:
		return false, fmt.Errorf("Invalid Credentials")
	default:
		contents, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("Error while calling bosh: %s", contents)
	}

}
