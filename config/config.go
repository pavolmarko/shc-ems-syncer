package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type ConfigJson struct {
	ShcHostport           string `json:"shc-hostport"`
	ShcIssuingCaFile      string `json:"shc-issuing-ca-file"`
	ShcClientKeyFile      string `json:"shc-client-key-file"`
	ShcClientCertFile     string `json:"shc-client-cert-file"`
	EmsEspHostport        string `json:"ems-esp-hostport"`
	EmsEspAccessTokenFile string `json:"ems-esp-access-token-file"`
}

type Config struct {
	ShcHostport       string
	EmsEspHostport    string
	EmsEspAccessToken string
}

func Read(configPath string) (Config, error) {
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("can't read config '%s': %v", configPath, err)
	}

	var cfgRaw ConfigJson
	if err := json.Unmarshal(bytes, &cfgRaw); err != nil {
		return Config{}, fmt.Errorf("can't parse config in `%s`: %v", configPath, err)
	}

	emsEspAccessToken, err := readAsString(cfgRaw.EmsEspAccessTokenFile)
	if err != nil {
		return Config{}, err
	}

	return Config{
		ShcHostport:       cfgRaw.ShcHostport,
		EmsEspHostport:    cfgRaw.EmsEspHostport,
		EmsEspAccessToken: emsEspAccessToken,
	}, nil
}

func readAsString(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("can't read '%s': %v", path, err)
	}

	return string(bytes), nil
}
