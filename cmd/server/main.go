package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/as283-ua/yappa/internal/server"
)

var (
	addr   *string
	cert   *string
	key    *string
	caCert *string
)

func main() {
	addr = flag.String("ip", "0.0.0.0:4433", "Host IP and port")
	cert = flag.String("cert", "certs/server/server.crt", "TLS Certificate")
	key = flag.String("key", "certs/server/server.key", "TLS Key")
	caCert = flag.String("ca", "certs/ca/ca.crt", "CA certificate")

	flag.Parse()

	server, err := server.SetupServer(&server.CmdArgs{
		Addr:   *addr,
		Cert:   *cert,
		Key:    *key,
		CaCert: *caCert,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server started on https://" + *addr)

	if err := server.ListenAndServeTLS(*cert, *key); err != nil {
		panic(err)
	}
}
