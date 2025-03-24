package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/as283-ua/yappa/internal/ca/handler"
	"github.com/as283-ua/yappa/internal/ca/middleware"
	"github.com/quic-go/quic-go/http3"
)

var (
	addr       *string
	cert       *string
	serverCert *string
	key        *string
	rootCa     *string
)

func getHashCert() ([]byte, error) {
	content, err := os.ReadFile(*serverCert)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(content)

	return hash[:], nil
}

func main() {
	addr = flag.String("ip", "0.0.0.0:4434", "Host IP and port")
	cert = flag.String("cert", "certs/ca_server/ca_server.crt", "TLS Certificate")
	key = flag.String("key", "certs/ca_server/ca_server.key", "TLS Key")
	serverCert = flag.String("server-cert", "certs/server/server.crt", "TLS Certificate for chat server")
	rootCa = flag.String("ca", "certs/ca/ca.crt", "Root CA certificate")

	serverCertHash, err := getHashCert()

	if err != nil {
		log.Fatal(err)
		return
	}

	router := http.NewServeMux()

	router.Handle("POST /allow/{username}", middleware.IsChatServer(serverCertHash, http.HandlerFunc(handler.AllowUser)))
	router.Handle("POST /sign/{username}", http.HandlerFunc(handler.SignCert))
	router.Handle("GET /certificates", http.HandlerFunc(handler.Getcertificates))
	router.Handle("POST /revoke/{username}", http.HandlerFunc(handler.Revoke))
	router.Handle("POST /reinstate/{username}", http.HandlerFunc(handler.Reinstate))

	fmt.Println("CA Server started on https://" + *addr)

	server := &http3.Server{
		Addr:      *addr,
		Handler:   router,
		TLSConfig: TLSConfig(),
	}

	if err := server.ListenAndServeTLS(*cert, *key); err != nil {
		log.Fatal(err)
	}
}

func TLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		log.Fatal(err)
	}

	rootCAs := x509.NewCertPool()
	caCertPath := *rootCa

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatal("Failed to read root CA certificate:", err)
	}

	if !rootCAs.AppendCertsFromPEM(caCert) {
		log.Fatal("Failed to append root CA certificate")
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		RootCAs:      rootCAs,
	}
}
