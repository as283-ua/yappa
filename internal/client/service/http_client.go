package service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

var httpClient *http.Client

func GetHttp3Client() (*http.Client, error) {
	if httpClient != nil {
		return nil, fmt.Errorf("no http client set up")
	}

	return httpClient, nil
}

func InitHttp3Client(caCertPath string) error {
	if httpClient != nil {
		return fmt.Errorf("no http client set up")
	}

	rootCAs := x509.NewCertPool()

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return err
	}

	rootCAs.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		RootCAs:    rootCAs,
		NextProtos: []string{"h3"},
	}

	transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig:      &quic.Config{},
	}

	httpClient = &http.Client{
		Transport: transport,
	}

	return nil
}

func UseCertificate(cert, key string) error {
	if httpClient == nil {
		return fmt.Errorf("no http client set up")
	}

	x509cert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}

	t, ok := httpClient.Transport.(*http3.Transport)

	if !ok {
		return fmt.Errorf("http transport error")
	}

	t.TLSClientConfig.Certificates = append(t.TLSClientConfig.Certificates, x509cert)

	return nil
}
