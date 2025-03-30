package common

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

var HttpClient *http.Client

func InitHttp3Client(caCertPath string) error {
	rootCAs := x509.NewCertPool()

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read root CA certificate: %v", err)
	}

	if !rootCAs.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to append root CA certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs:    rootCAs,
		NextProtos: []string{"h3"},
	}

	transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig: &quic.Config{
			EnableDatagrams: true,
		},
	}

	transport.EnableDatagrams = true

	HttpClient = &http.Client{
		Transport: transport,
	}
	return nil
}

func AddTlsCert(cert, key string) error {
	tlsCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}
	tlsConfig := HttpClient.Transport.(*http3.Transport).TLSClientConfig
	tlsConfig.Certificates = append(tlsConfig.Certificates, tlsCert)

	return nil
}
