// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package temporaltest

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

type certData struct {
	cert  *x509.Certificate
	bytes []byte
	key   *rsa.PrivateKey
}

type Certificate struct {
	CertPem []byte
	KeyPem  []byte
}

type Bundle struct {
	Ca     *Certificate
	Client *Certificate
	Server *Certificate
}

func GenerateCertificates() (*Bundle, error) {
	ca, err := generateCa()
	if err != nil {
		return nil, err
	}

	server, err := generateServerCertificate(ca)
	if err != nil {
		return nil, err
	}
	client, err := generateClientCertificate(ca)
	if err != nil {
		return nil, err
	}

	b := &Bundle{
		Ca:     convertCertData(ca),
		Client: convertCertData(client),
		Server: convertCertData(server),
	}

	return b, nil
}

func convertCertData(d *certData) *Certificate {
	certPem := new(bytes.Buffer)
	pem.Encode(certPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: d.bytes,
	})

	priv := new(bytes.Buffer)
	pem.Encode(priv, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(d.key),
	})

	return &Certificate{
		CertPem: certPem.Bytes(),
		KeyPem:  priv.Bytes(),
	}
}

func generateCa() (*certData, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2022),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	private, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ca key: %w", err)
	}

	bytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &private.PublicKey, private)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ca certificate: %w", err)
	}

	return &certData{
		cert:  ca,
		bytes: bytes,
		key:   private,
	}, nil
}

func generateServerCertificate(ca *certData) (*certData, error) {
	return generateCertificate(ca, func(cert *x509.Certificate) {
		cert.DNSNames = []string{"localhost"}
		cert.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	})
}

func generateClientCertificate(ca *certData) (*certData, error) {
	return generateCertificate(ca, func(cert *x509.Certificate) {
		cert.DNSNames = []string{"client"}
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	})
}

func generateCertificate(ca *certData, apply func(*x509.Certificate)) (*certData, error) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2022),
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

	apply(cert)
	private, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate key: %w", err)
	}

	bytes, err := x509.CreateCertificate(rand.Reader, cert, ca.cert, &private.PublicKey, ca.key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ca certificate: %w", err)
	}

	return &certData{
		cert:  cert,
		bytes: bytes,
		key:   private,
	}, nil
}
