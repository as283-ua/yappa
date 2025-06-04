package ca

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"
	"net/http"
	"os"

	"github.com/as283-ua/yappa/internal/ca/logging"
	"github.com/as283-ua/yappa/internal/ca/settings"
	"github.com/as283-ua/yappa/internal/ca/signature"
	"github.com/quic-go/quic-go/http3"
)

var (
	caCert *x509.Certificate
	caKey  any
)

var (
	tlsConfig *tls.Config
)

var logger *log.Logger

func SetupServer(cmdArgs *settings.CaCfg) (*http3.Server, error) {
	settings.CaSettings = cmdArgs
	err := settings.CaSettings.Validate()
	if err != nil {
		return nil, err
	}

	tlsConfig = getTlsConfig()

	serverCertSerial, err := getCertSerialN(settings.CaSettings.Chat.Cert)

	if err != nil {
		return nil, err
	}

	if cmdArgs.Logs != "" {
		err = logging.SetOutput(cmdArgs.Logs)
		if err != nil {
			log.Fatal(err)
		}
	}
	logger = logging.GetLogger()

	router := http.NewServeMux()

	router.Handle("POST /allow", signature.MatchCertSerialNumber(serverCertSerial, http.HandlerFunc(signature.AllowUser)))
	router.Handle("POST /sign", http.HandlerFunc(signature.SignCert(caCert, caKey)))
	router.Handle("GET /certificates", http.HandlerFunc(signature.Getcertificates))
	router.Handle("POST /revoke/{username}", http.HandlerFunc(signature.Revoke))
	router.Handle("POST /reinstate/{username}", http.HandlerFunc(signature.Reinstate))

	return &http3.Server{
		Addr:      settings.CaSettings.Addr,
		Handler:   router,
		TLSConfig: tlsConfig,
	}, nil
}

func getTlsConfig() *tls.Config {
	err := loadCA()

	if err != nil {
		logger.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair(settings.CaSettings.Tls.Cert, settings.CaSettings.Tls.Key)
	if err != nil {
		logger.Fatal(err)
	}

	rootCAs := x509.NewCertPool()
	caCertPath := settings.CaSettings.Cacert

	caCertBytes, err := os.ReadFile(caCertPath)
	if err != nil {
		logger.Fatal("Failed to read root CA certificate:", err)
	}

	rootCAs.AppendCertsFromPEM(caCertBytes)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    rootCAs,
		ClientAuth:   tls.RequestClientCert,
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
	caCertBytes, err := os.ReadFile(settings.CaSettings.Cacert)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(caCertBytes)

	caCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}

	caKeyBytes, err := os.ReadFile(settings.CaSettings.Key)
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
