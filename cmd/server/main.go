package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/as283-ua/yappa/internal/server/handler"
	"github.com/quic-go/quic-go/http3"
)

var (
	addr *string
	cert *string
	key  *string
)

func main() {
	addr = flag.String("ip", "0.0.0.0:4433", "Host IP and port")
	cert = flag.String("cert", "certs/server/server.crt", "TLS Certificate")
	key = flag.String("key", "certs/server/server.key", "TLS Key")

	router := http.NewServeMux()

	router.Handle("CONNECT /connect", http.HandlerFunc(handler.HandleConnection))

	fmt.Println("Server started on https://" + *addr)

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
