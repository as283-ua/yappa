package ca

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"

	"github.com/as283-ua/yappa/internal/ca/handler"
	"github.com/as283-ua/yappa/internal/middleware"
	"github.com/quic-go/quic-go/http3"
)

var (
	caCert *x509.Certificate
	caKey  any
)

type CmdArgs struct {
	Addr           string
	Cert           string
	Key            string
	ChatServerCert string
	RootCa         string
	CaKey          string
}

var (
	args      *CmdArgs
	tlsConfig *tls.Config
)

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

	if c.ChatServerCert == "" {
		return fmt.Errorf("chatServerCert must not be empty")
	}

	if c.RootCa == "" {
		return fmt.Errorf("rootCa must not be empty")
	}

	if c.CaKey == "" {
		return fmt.Errorf("caKey must not be empty")
	}

	return nil
}

func SetupServer(cmdArgs *CmdArgs) (*http3.Server, error) {
	args = cmdArgs
	err := args.Validate()
	if err != nil {
		return nil, err
	}

	tlsConfig = getTlsConfig()

	serverCertSerial, err := getCertSerialN(args.ChatServerCert)

	if err != nil {
		return nil, err
	}

	router := http.NewServeMux()

	router.Handle("POST /allow", middleware.MatchCertSerialNumber(serverCertSerial, http.HandlerFunc(handler.AllowUser)))
	router.Handle("POST /sign", http.HandlerFunc(handler.SignCert(caCert, caKey)))
	router.Handle("GET /certificates", http.HandlerFunc(handler.Getcertificates))
	router.Handle("POST /revoke/{username}", http.HandlerFunc(handler.Revoke))
	router.Handle("POST /reinstate/{username}", http.HandlerFunc(handler.Reinstate))

	return &http3.Server{
		Addr:      args.Addr,
		Handler:   router,
		TLSConfig: tlsConfig,
	}, nil
}

func getTlsConfig() *tls.Config {
	err := loadCA()

	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair(args.Cert, args.Key)
	if err != nil {
		log.Fatal(err)
	}

	rootCAs := x509.NewCertPool()
	caCertPath := args.RootCa

	caCertBytes, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatal("Failed to read root CA certificate:", err)
	}

	rootCAs.AppendCertsFromPEM(caCertBytes)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    rootCAs,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		NextProtos:   []string{"h3"},
	}
}

func getCertSerialN(serverCert string) (*big.Int, error) {
	content, err := os.ReadFile(serverCert)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(content)
	cert, err := x509.ParseCertificate(block.Bytes)

	if err != nil {
		return nil, err
	}

	return cert.SerialNumber, nil
}

func loadCA() error {
	caCertBytes, err := os.ReadFile(args.RootCa)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(caCertBytes)

	caCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}

	caKeyBytes, err := os.ReadFile(args.CaKey)
	if err != nil {
		return err
	}

	block, _ = pem.Decode(caKeyBytes)

	caKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	return nil
}
