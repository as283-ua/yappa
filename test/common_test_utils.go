package test

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func GetHttp3Client(certPath, certificateOwner, caCertPath string) *http.Client {
	rootCAs := x509.NewCertPool()

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
		cert, err := tls.LoadX509KeyPair(certPath+"/"+certificateOwner+"/"+certificateOwner+".crt", certPath+"/"+certificateOwner+"/"+certificateOwner+".key")
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
