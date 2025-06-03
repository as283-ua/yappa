package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/as283-ua/yappa/internal/ca"
	"github.com/as283-ua/yappa/internal/ca/logging"
	"github.com/as283-ua/yappa/internal/ca/settings"
)

func main() {
	addr := flag.String("ip", "0.0.0.0:4434", "Host IP and port")
	cert := flag.String("cert", "certs/ca_tls/ca_tls.crt", "TLS Certificate")
	key := flag.String("key", "certs/ca_tls/ca_tls.key", "TLS Key")
	chatServerCert := flag.String("server-cert", "certs/server/server.crt", "TLS Certificate for chat server")
	rootCa := flag.String("ca", "certs/ca/ca.crt", "Root CA certificate")
	caKey := flag.String("ca-key", "certs/ca/ca.key", "Root CA private key")
	logDir := flag.String("logs", "logs/ca/", "Log directory")

	flag.Parse()

	server, err := ca.SetupServer(&settings.CaCfg{
		Addr:           *addr,
		Cert:           *cert,
		Key:            *key,
		ChatServerCert: *chatServerCert,
		RootCa:         *rootCa,
		CaKey:          *caKey,
		LogDir:         *logDir,
	})

	log := logging.GetLogger()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		server.Close()
		log.Println("Closed server")
		os.Exit(0)
	}()

	if err != nil {
		log.Fatal("Error seting up server:", err)
	}

	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")

	fmt.Println("CA Server started on https://" + *addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
