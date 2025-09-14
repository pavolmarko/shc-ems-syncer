package ems

import (
	"fmt"
	"io"
	"net/http"
)

type EmsClient struct {
	hostport    string
	accessToken string

	httpClient *http.Client
}

func CreateClient(hostport string, accessToken string) *EmsClient {
	return &EmsClient{
		hostport:   hostport,
		httpClient: &http.Client{},
	}
}

func (cli *EmsClient) Ping() (string, error) {
	url := cli.urlFor("api/thermostat/seltemp")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("can't init req to '%s': %v", url, err)
	}

	req.Header.Add("Authorization", "Bearer: "+cli.accessToken)

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET request to '%s' failed: %v", url, err)
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("can't read response from '%s': %v", url, err)
	}

	return string(bytes), nil

}

func (cli *EmsClient) urlFor(part string) string {
	return fmt.Sprintf("http://%s/%s", cli.hostport, part)
}
