package shc

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const SHC_PORT_PUBLIC = 8446
const SHC_PORT_CLIENT_MGMT = 8443

type ShcClient struct {
	host       string
	caCertPool *x509.CertPool
	clientCert tls.Certificate

	httpClient     *http.Client
	mtlsHttpClient *http.Client // Will include client cert
}

func CreateClient(host string, caCertPool *x509.CertPool, clientCert tls.Certificate) *ShcClient {
	cli := &ShcClient{
		host:       host,
		caCertPool: caCertPool,
		clientCert: clientCert,
	}

	tlsClientConfig := &tls.Config{
		InsecureSkipVerify:    true, // VerifyPeerCertificate will do this
		RootCAs:               caCertPool,
		VerifyPeerCertificate: cli.VerifyServerCert,
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsClientConfig
	cli.httpClient = &http.Client{
		Transport: transport,
	}

	mtlsClientConfig := transport.TLSClientConfig.Clone()
	mtlsClientConfig.Certificates = []tls.Certificate{clientCert}

	mtlsTransport := http.DefaultTransport.(*http.Transport).Clone()
	mtlsTransport.TLSClientConfig = mtlsClientConfig
	cli.mtlsHttpClient = &http.Client{
		Transport: mtlsTransport,
	}

	return cli
}

// Verifying on our own because the server certificate of SHC usually doesn't have a hostname.
func (cli *ShcClient) VerifyServerCert(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no certificates presented")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("can't parse leaf cert: %v", err)
	}

	// Not setting `DNSName` on purpose here to avoid verifying the hostname.
	opts := x509.VerifyOptions{
		Intermediates: x509.NewCertPool(),
		Roots:         cli.caCertPool,
	}

	for i, cert := range rawCerts {
		if i == 0 {
			continue
		}

		crt, err := x509.ParseCertificate(cert)
		if err != nil {
			return fmt.Errorf("can't parse intermediate cert: %v", err)
		}

		opts.Intermediates.AddCert(crt)
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("server cert verification failed: %v", err)
	}

	fmt.Println("successfully verified")

	return nil
}

func (cli *ShcClient) Ping() (string, error) {
	url := cli.publicUrlFor("smarthome/public/information")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("can't init req to '%s': %v", url, err)
	}

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

func (cli *ShcClient) FormatClientCertForRegister() string {
	// This is pretty weird, but the following BoschSmartHome doc says to do it:
	// https://github.com/BoschSmartHome/bosch-shc-api-docs/blob/4f6ecf0fadd3d3e855d81f7f413891c9fd07a3bd/postman/README.md#customize-the-certificate
	// Usually we'd use "encoding/pem"'s pem.EncodeToMemory, but the special handling makes it easier to just manually base64.

	specialHeader := "-----BEGIN CERTIFICATE-----\\r"
	specialFooter := "\\r-----BEGIN CERTIFICATE-----"
	bodyWithoutNewlines := base64.StdEncoding.EncodeToString(cli.clientCert.Leaf.Raw)

	return specialHeader + bodyWithoutNewlines + specialFooter
}

func (cli *ShcClient) Register(systemPassword string) (string, error) {
	systemPasswordEncoded := base64.StdEncoding.EncodeToString([]byte(systemPassword))

	body := registerMsg{
		Id:          "oss_shc_ems_syncer",
		Name:        "OSS github.com/pavolmarko/shc-ems-syncer",
		PrimaryRole: "ROLE_RESTRICTED_CLIENT",
		Certificate: cli.FormatClientCertForRegister(),
	}

	bodyEncoded := new(bytes.Buffer)
	if err := json.NewEncoder(bodyEncoded).Encode(body); err != nil {
		return "", fmt.Errorf("can't encode registry request body: %v", err)
	}

	url := cli.clientMgmtUrlFor("smarthome/clients")
	req, err := http.NewRequest("POST", url, bodyEncoded)
	if err != nil {
		return "", fmt.Errorf("can't init req to '%s': %v", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Systempassword", systemPasswordEncoded)

	resp, err := cli.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("'register' POST request to '%s' failed: %v", url, err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("can't read response from '%s': %v", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SHC register request filed with '%s' - body: %s", resp.Status, string(respBody))
	}

	return string(respBody), nil
}

func (cli *ShcClient) publicUrlFor(part string) string {
	return fmt.Sprintf("https://%s:%d/%s", cli.host, SHC_PORT_PUBLIC, part)
}

func (cli *ShcClient) clientMgmtUrlFor(part string) string {
	return fmt.Sprintf("https://%s:%d/%s", cli.host, SHC_PORT_CLIENT_MGMT, part)
}
