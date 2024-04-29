// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
)

// certSetup creates client & server certs and TLS config for testing.
func certSetup(dir string) (serverTLSConf *tls.Config, clientTLSConf *tls.Config) {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(1234567890),
		Subject:               pkix.Name{Organization: []string{"Korrel8r Project"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	test.PanicErr(err)

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	test.PanicErr(err)

	// pem encode
	caPEM := new(bytes.Buffer)
	test.PanicErr(pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}))

	caPrivKeyPEM := new(bytes.Buffer)
	test.PanicErr(pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	}))

	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject:      pkix.Name{Organization: []string{"Korrel8r Project"}},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	test.PanicErr(err)

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	test.PanicErr(err)

	certPEM := new(bytes.Buffer)
	test.PanicErr(pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}))

	certPrivKeyPEM := new(bytes.Buffer)
	test.PanicErr(pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	}))

	test.PanicErr(os.WriteFile(filepath.Join(dir, "tls.crt"), certPEM.Bytes(), 0444))
	test.PanicErr(os.WriteFile(filepath.Join(dir, "tls.key"), certPrivKeyPEM.Bytes(), 0444))

	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	test.PanicErr(err)

	serverTLSConf = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPEM.Bytes())
	clientTLSConf = &tls.Config{
		RootCAs: certpool,
	}

	return serverTLSConf, clientTLSConf
}
