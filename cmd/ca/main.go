package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/as283-ua/yappa/internal/ca/handler"
	"github.com/quic-go/quic-go/http3"
)

var (
	addr *string
	cert *string
	key  *string
)

func main() {
	addr = flag.String("ip", "0.0.0.0:4434", "Host IP and port")
	cert = flag.String("cert", "certs/ca/ca.crt", "TLS Certificate")
	key = flag.String("key", "certs/ca/ca.key", "TLS Key")

	router := http.NewServeMux()

	router.Handle("POST /allow/{username}", http.HandlerFunc(handler.AllowUser))
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
		panic(err)
	}
}

func TLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		log.Fatal(err)
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}
}
