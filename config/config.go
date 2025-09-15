package config

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
)

type ConfigJson struct {
	ShcHost               string `json:"shc-host"`
	ShcIssuingCaFile      string `json:"shc-issuing-ca-file"`
	ShcClientKeyFile      string `json:"shc-client-key-file"`
	ShcClientCertFile     string `json:"shc-client-cert-file"`
	EmsEspHostport        string `json:"ems-esp-hostport"`
	EmsEspAccessTokenFile string `json:"ems-esp-access-token-file"`
}

type Config struct {
	ShcHost           string
	EmsEspHostport    string
	EmsEspAccessToken string
	ShcCaCertPool     *x509.CertPool
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

	bytes, err = os.ReadFile(cfgRaw.ShcIssuingCaFile)
	if err != nil {
		return Config{}, fmt.Errorf("can't read issuing ca file '%s': %v", cfgRaw.ShcIssuingCaFile, err)
	}

	shcCaCertPool := x509.NewCertPool()
	if ok := shcCaCertPool.AppendCertsFromPEM(bytes); !ok {
		return Config{}, fmt.Errorf("no certs found in %s", cfgRaw.ShcIssuingCaFile)
	}

	return Config{
		ShcHost:           cfgRaw.ShcHost,
		EmsEspHostport:    cfgRaw.EmsEspHostport,
		EmsEspAccessToken: emsEspAccessToken,
		ShcCaCertPool:     shcCaCertPool,
	}, nil
}

func readAsString(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("can't read '%s': %v", path, err)
	}

	return string(bytes), nil
}
