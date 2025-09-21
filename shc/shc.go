package shc

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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

	// A HTTP client which doesn't send a client certificate. Used for "public" endpoint.
	httpClientPublic *http.Client
	// A HTTP client which actively fails with a useful-ish error message if
	// a client certificate has been requested. Used for "register", see
	// NoClientCertForRegister function
	httpClientRegister *http.Client
	// For regular operation after register. will include client cert.
	httpClientMain *http.Client
}

func CreateClient(host string, caCertPool *x509.CertPool, clientCert tls.Certificate) *ShcClient {
	cli := &ShcClient{
		host:       host,
		caCertPool: caCertPool,
		clientCert: clientCert,
	}

	// For the public endpoint, perform custom CA verification and don't expect
	// to be requested for the client cert
	cli.httpClientPublic = cli.createHttpClient(cli.NoClientCertForPublic)

	// For the "register" endpoint, perform custom CA verification and fail with special error message if requested for client cert
	cli.httpClientRegister = cli.createHttpClient(cli.NoClientCertForRegister)

	// For the regular operation, send our client cert
	cli.httpClientMain = cli.createHttpClient(cli.SendClientCert)

	return cli
}

func (cli *ShcClient) createHttpClient(getClientCertFunc func(*tls.CertificateRequestInfo) (*tls.Certificate, error)) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify:    true, // VerifyPeerCertificate will do this
		VerifyPeerCertificate: cli.VerifyServerCert,
		GetClientCertificate:  getClientCertFunc,
	}

	return &http.Client{
		Transport: transport,
	}
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

	fmt.Println("successfully verified SHC server cert")

	return nil
}

// Sending our client cert unconditionally because the request from SHC is strange.
func (cli *ShcClient) SendClientCert(req *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	if err := req.SupportsCertificate(&cli.clientCert); err != nil {
		return nil, fmt.Errorf("this client seems to be unknown to SHC - have you registered? Detail: %v", err)
	}

	return &cli.clientCert, nil
}

// Expecting to not need a clinet cert for the public endpoint
func (cli *ShcClient) NoClientCertForPublic(req *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return nil, fmt.Errorf("unexpected client certificate request for public endpoint")
}

// If SHC asks for a client cert on "register", it is likely not in registration mode.
func (cli *ShcClient) NoClientCertForRegister(req *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return nil, fmt.Errorf("SHC asked for a client certificate - have you pressed the button on SHC??")
}

func (cli *ShcClient) Ping() (string, error) {
	url := cli.publicUrlFor("smarthome/public/information")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("can't init req to '%s': %v", url, err)
	}

	resp, err := cli.httpClientPublic.Do(req)
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

func (cli *ShcClient) formatClientCertForRegister() string {
	// The following BoschSmartHome doc says to do something weird:
	// https://github.com/BoschSmartHome/bosch-shc-api-docs/blob/4f6ecf0fadd3d3e855d81f7f413891c9fd07a3bd/postman/README.md#customize-the-certificate
	// But that actually didn't work. Regular PEM encoding ended up working.

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cli.clientCert.Leaf.Raw,
	}

	return string(pem.EncodeToMemory(block))
}

func (cli *ShcClient) Register(systemPassword string) (string, error) {
	systemPasswordEncoded := base64.StdEncoding.EncodeToString([]byte(systemPassword))

	body := registerMsg{
		Type:        "client",
		Id:          "oss_shc_ems_syncer",
		Name:        "OSS SHC EMS Syncer",
		PrimaryRole: "ROLE_RESTRICTED_CLIENT",
		Certificate: cli.formatClientCertForRegister(),
	}

	bodyEncoded := new(bytes.Buffer)
	if err := json.NewEncoder(bodyEncoded).Encode(body); err != nil {
		return "", fmt.Errorf("can't encode registry request body: %v", err)
	}

	fmt.Println("body:")
	fmt.Println(bodyEncoded.String())

	url := cli.clientMgmtUrlFor("smarthome/clients")
	req, err := http.NewRequest("POST", url, bodyEncoded)
	if err != nil {
		return "", fmt.Errorf("can't init req to '%s': %v", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Systempassword", systemPasswordEncoded)

	resp, err := cli.httpClientRegister.Do(req)
	if err != nil {
		return "", fmt.Errorf("'register' POST request to '%s' failed: %v", url, err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("can't read response from '%s': %v", url, err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("SHC register request failed with '%s' - body: %s", resp.Status, string(respBody))
	}

	return string(respBody), nil
}

func (cli *ShcClient) publicUrlFor(part string) string {
	return fmt.Sprintf("https://%s:%d/%s", cli.host, SHC_PORT_PUBLIC, part)
}

func (cli *ShcClient) clientMgmtUrlFor(part string) string {
	return fmt.Sprintf("https://%s:%d/%s", cli.host, SHC_PORT_CLIENT_MGMT, part)
}
