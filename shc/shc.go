package shc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
)

const SHC_PORT_PUBLIC = 8446
const SHC_PORT_PRIVATE = 8444

type ShcClient struct {
	host string

	caCertPool *x509.CertPool
	httpClient *http.Client
}

func CreateClient(host string, caCertPool *x509.CertPool) *ShcClient {
	cli := &ShcClient{
		host:       host,
		caCertPool: caCertPool,
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify:    true, // VerifyPeerCertificate will do this
		RootCAs:               caCertPool,
		VerifyPeerCertificate: cli.VerifyServerCert,
	}

	cli.httpClient = &http.Client{
		Transport: customTransport,
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

func (cli *ShcClient) publicUrlFor(part string) string {
	return fmt.Sprintf("https://%s:%d/%s", cli.host, SHC_PORT_PUBLIC, part)
}
