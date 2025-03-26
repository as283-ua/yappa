package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/as283-ua/yappa/internal/ca"
)

var (
	addr           *string
	cert           *string
	chatServerCert *string
	key            *string
	rootCa         *string
	caKey          *string
)

func main() {
	addr = flag.String("ip", "0.0.0.0:4434", "Host IP and port")
	cert = flag.String("cert", "certs/ca_server/ca_server.crt", "TLS Certificate")
	key = flag.String("key", "certs/ca_server/ca_server.key", "TLS Key")
	chatServerCert = flag.String("server-cert", "certs/server/server.crt", "TLS Certificate for chat server")
	rootCa = flag.String("ca", "certs/ca/ca.crt", "Root CA certificate")
	caKey = flag.String("ca-key", "certs/ca/ca.key", "Root CA private key")

	flag.Parse()

	server, err := ca.SetupServer(&ca.CmdArgs{
		Addr:           *addr,
		Cert:           *cert,
		Key:            *key,
		ChatServerCert: *chatServerCert,
		RootCa:         *rootCa,
		CaKey:          *caKey,
	})

	if err != nil {
		log.Fatal("Error seting up server:", err)
	}

	fmt.Println("CA Server started on https://" + *addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
