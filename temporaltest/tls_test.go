// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/DataDog/temporalite/internal/examples/helloworld"
	"github.com/DataDog/temporalite/temporaltest"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/config"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"os"
	"testing"
	"time"
)

type certData struct {
	cert  *x509.Certificate
	bytes []byte
	key   *ecdsa.PrivateKey
}

type certificate struct {
	CertPEM []byte
	KeyPEM  []byte
}

type bundle struct {
	Client *certificateAndCAPair
	Server *certificateAndCAPair
}

type certificateAndCAPair struct {
	Authority *certificate
	Cert      *certificate
}

func TestNewServerWithMutualTLS(t *testing.T) {
	testNewServerWithTLSEnabled(t, true, "test_mtls")
}

func TestNewServerWithTLS(t *testing.T) {
	testNewServerWithTLSEnabled(t, false, "test_tls")
}

func testNewServerWithTLSEnabled(t *testing.T, useMutualTls bool, taskQueue string) {
	b, err := generateCertificates()
	if err != nil {
		t.Fatal(err)
	}

	kp, err := tls.X509KeyPair(b.Client.Cert.CertPEM, b.Client.Cert.KeyPEM)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(b.Server.Authority.CertPEM)
	// pool.AppendCertsFromPEM(b.Client.Authority.CertPEM)

	cfg := config.Config{
		Global: config.Global{
			TLS: config.RootTLS{
				Frontend: config.GroupTLS{
					Client: config.ClientTLS{
						RootCAData: []string{base64.StdEncoding.EncodeToString(b.Server.Authority.CertPEM)},
					},
					Server: config.ServerTLS{
						CertData: base64.StdEncoding.EncodeToString(b.Server.Cert.CertPEM),
						KeyData:  base64.StdEncoding.EncodeToString(b.Server.Cert.KeyPEM),
						ClientCAData: []string{
							base64.StdEncoding.EncodeToString(b.Client.Authority.CertPEM),
							base64.StdEncoding.EncodeToString(b.Server.Authority.CertPEM)},
						RequireClientAuth: useMutualTls,
					},
				},
			},
		},
	}

	if useMutualTls {
		cfg.Global.TLS.Internode = config.GroupTLS{
			Client: config.ClientTLS{
				RootCAData: []string{base64.StdEncoding.EncodeToString(b.Server.Authority.CertPEM)},
			},
			Server: config.ServerTLS{
				CertData:          base64.StdEncoding.EncodeToString(b.Server.Cert.CertPEM),
				KeyData:           base64.StdEncoding.EncodeToString(b.Server.Cert.KeyPEM),
				ClientCAData:      []string{base64.StdEncoding.EncodeToString(b.Server.Authority.CertPEM)},
				RequireClientAuth: useMutualTls,
			},
		}
	}

	configBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	file, err := ioutil.TempFile("", "config")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(file.Name())
	_, err = file.Write(configBytes)
	if err != nil {
		t.Fatal(err)
	}

	ts := temporaltest.NewServer(
		temporaltest.WithConfigFile(file.Name()),
		temporaltest.WithClientOptions(client.Options{
			ConnectionOptions: client.ConnectionOptions{
				TLS: &tls.Config{
					Certificates: []tls.Certificate{kp},
					RootCAs:      pool,
				},
			},
		}),
		temporaltest.WithT(t))

	c := ts.Client()

	ts.Worker(taskQueue, func(registry worker.Registry) {
		helloworld.RegisterWorkflowsAndActivities(registry)
	})

	// Start test workflow
	wfr, err := c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{TaskQueue: taskQueue},
		helloworld.Greet,
		"world",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Get workflow result
	var result string
	if err := wfr.Get(context.Background(), &result); err != nil {
		t.Fatal(err)
	}

	// Print result
	fmt.Println(result)
	// Output: Hello world

	if result != "Hello world" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func generateCertificates() (*bundle, error) {
	serverAuthority, serverAuthorityCertificate, err := generateCA("server")
	if err != nil {
		return nil, err
	}

	server, err := generateClientOrServerCertificate(serverAuthority, true)
	if err != nil {
		return nil, err
	}

	clientAuthority, clientAuthorityCertificate, err := generateCA("client")
	if err != nil {
		return nil, err
	}

	clientCertificate, err := generateClientOrServerCertificate(clientAuthority, false)
	if err != nil {
		return nil, err
	}

	b := &bundle{
		Client: &certificateAndCAPair{
			Authority: clientAuthorityCertificate,
			Cert:      clientCertificate,
		},
		Server: &certificateAndCAPair{
			Authority: serverAuthorityCertificate,
			Cert:      server,
		},
	}

	return b, nil
}

func convertCertData(d *certData) (*certificate, error) {

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: d.bytes})
	keyPKCS, err := x509.MarshalECPrivateKey(d.key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyPKCS,
	})

	return &certificate{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
	}, nil
}

func generateCA(name string) (*certData, *certificate, error) {
	ca := &x509.Certificate{
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Company, INC."},
			OrganizationalUnit: nil,
			Locality:           []string{"San Francisco"},
			Province:           []string{""},
			StreetAddress:      []string{"Golden Gate Bridge"},
			PostalCode:         []string{"94016"},
			CommonName:         name,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	ca.SerialNumber, _ = rand.Int(rand.Reader, big.NewInt(math.MaxInt64))

	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ca key: %w", err)
	}

	b, err := x509.CreateCertificate(rand.Reader, ca, ca, &private.PublicKey, private)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ca certificate: %w", err)
	}

	data := &certData{
		cert:  ca,
		bytes: b,
		key:   private,
	}

	if err != nil {
		return nil, nil, err
	}

	conv, err := convertCertData(data)
	if err != nil {
		return nil, nil, err
	}

	return data, conv, err
}

func generateClientOrServerCertificate(ca *certData, isServer bool) (*certificate, error) {
	var err error
	var data *certData
	if isServer {
		data, err = generateCertificate(ca, func(cert *x509.Certificate) {
			cert.DNSNames = []string{"localhost"}
			cert.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
			cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
		})
	} else {
		data, err = generateCertificate(ca, func(cert *x509.Certificate) {
			cert.DNSNames = []string{"client"}
			cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		})
	}

	if err != nil {
		return nil, err
	}

	conv, err := convertCertData(data)
	if err != nil {
		return nil, err
	}

	return conv, nil
}

func generateCertificate(ca *certData, apply func(*x509.Certificate)) (*certData, error) {
	cert := &x509.Certificate{
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},

		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	cert.SerialNumber, _ = rand.Int(rand.Reader, big.NewInt(math.MaxInt64))

	apply(cert)
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate key: %w", err)
	}

	b, err := x509.CreateCertificate(rand.Reader, cert, ca.cert, private.Public(), ca.key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ca certificate: %w", err)
	}

	return &certData{
		cert:  cert,
		bytes: b,
		key:   private,
	}, nil
}
