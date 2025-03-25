package test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func getHttp3Client(certificateOwner string) *http.Client {
	rootCAs := x509.NewCertPool()
	caCertPath := "../certs/ca/ca.crt"

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatal("Failed to read root CA certificate:", err)
	}

	rootCAs.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		RootCAs:    rootCAs,
		NextProtos: []string{"h3"},
	}

	if certificateOwner != "" {
		cert, err := tls.LoadX509KeyPair("../certs/"+certificateOwner+"/"+certificateOwner+".crt", "../certs/"+certificateOwner+"/"+certificateOwner+".key")
		if err != nil {
			log.Fatal(err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig:      &quic.Config{},
	}

	return &http.Client{
		Transport: transport,
	}
}

func TestAllowNoCert(t *testing.T) {
	client := getHttp3Client("")
	_, err := client.Post("https://yappa.io:4434/allow/user1", "text/plain", bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	if err == nil {
		t.Error("Not providing a certificate should give an error")
	}
}

func TestAllowServerCert(t *testing.T) {
	client := getHttp3Client("server")
	resp, err := client.Post("https://yappa.io:4434/allow/user2", "text/plain", bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("Status should be OK 200")
	}
}

func TestAllowTestCert(t *testing.T) {
	client := getHttp3Client("test")
	resp, err := client.Post("https://yappa.io:4434/allow/user1", "text/plain", bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("Status should be unauthorized 401")
	}
}
