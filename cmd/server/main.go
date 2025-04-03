package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/as283-ua/yappa/internal/server"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/as283-ua/yappa/internal/server/settings"
)

func main() {
	addr := flag.String("addr", "0.0.0.0:4433", "Host IP and port")
	cert := flag.String("cert", "certs/server/server.crt", "TLS Certificate")
	key := flag.String("key", "certs/server/server.key", "TLS Key")
	caCert := flag.String("ca", "certs/ca/ca.crt", "CA certificate")
	caAddr := flag.String("ca-addr", "yappa.io:4434", "CA server ip address and port")
	logDir := flag.String("logs", "logs/serv/", "Log directory")

	flag.Parse()

	server, err := server.SetupServer(&settings.ChatCfg{
		Addr:   *addr,
		Cert:   *cert,
		Key:    *key,
		CaCert: *caCert,
		CaAddr: *caAddr,
		LogDir: *logDir,
	}, server.SetupPgxDb(context.Background()))

	log := logging.Logger

	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		server.Close()
		log.Println("Closed server")
		os.Exit(0)
	}()

	fmt.Println("Server started on https://" + *addr)

	if err := server.ListenAndServeTLS(*cert, *key); err != nil {
		log.Fatal(err)
	}
}
