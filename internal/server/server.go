package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/as283-ua/yappa/internal/server/handler"
	"github.com/quic-go/quic-go/http3"
)

type CmdArgs struct {
	Addr   string
	Cert   string
	Key    string
	CaCert string
}

func (c *CmdArgs) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address must not be empty")
	}

	if c.Cert == "" {
		return fmt.Errorf("cert must not be empty")
	}

	if c.Key == "" {
		return fmt.Errorf("key must not be empty")
	}

	return nil
}

var (
	args      *CmdArgs
	tlsConfig *tls.Config
)

func SetupServer(cmdArgs *CmdArgs) (*http3.Server, error) {
	args = cmdArgs
	err := args.Validate()

	if err != nil {
		return nil, err
	}

	tlsConfig, err = getTlsConfig()

	if err != nil {
		return nil, err
	}

	router := http.NewServeMux()

	router.Handle("CONNECT /connect", http.HandlerFunc(handler.HandleConnection))

	return &http3.Server{
		Addr:      args.Addr,
		Handler:   router,
		TLSConfig: tlsConfig,
	}, err
}

func getTlsConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(args.Cert, args.Key)
	if err != nil {
		return nil, err
	}

	rootCAs := x509.NewCertPool()
	caCertPath := args.CaCert

	caCertBytes, err := os.ReadFile(caCertPath)

	if err != nil {
		return nil, err
	}

	rootCAs.AppendCertsFromPEM(caCertBytes)

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		NextProtos:   []string{"h3"},
	}, nil
}
