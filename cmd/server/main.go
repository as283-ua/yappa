package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/as283-ua/yappa/internal/server"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/as283-ua/yappa/internal/server/settings"
)

func readCfgFile(path string) (*settings.ChatCfg, error) {
	if path == "" {
		return nil, errors.New("empty path passed to be read as config")
	}
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	cfgRaw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &settings.ChatCfg{}
	_, err = toml.Decode(string(cfgRaw), &cfg)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

var (
	addr    = flag.String("addr", "0.0.0.0:4433", "Host IP and port")
	cert    = flag.String("cert", "certs/server/server.crt", "TLS Certificate")
	key     = flag.String("key", "certs/server/server.key", "TLS Key")
	caCert  = flag.String("ca", "certs/ca/ca.crt", "CA certificate")
	caAddr  = flag.String("ca-addr", "yappa.io:4434", "CA server ip address and port")
	logDir  = flag.String("logs", "logs/serv/", "Log directory")
	cfgPath = flag.String("config", "cfg/yappad.toml", "Configuration file")
)

// apply default values or those specified explicitly through command line options
func applyCmdArgs(cfg *settings.ChatCfg) {
	explicitFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})

	if cfg.Addr == "" || explicitFlags["addr"] {
		cfg.Addr = *addr
	}
	if cfg.Logs == "" || explicitFlags["logs"] {
		cfg.Logs = *logDir
	}
	if cfg.Tls.Cert == "" || explicitFlags["cert"] {
		cfg.Tls.Cert = *cert
	}
	if cfg.Tls.Key == "" || explicitFlags["key"] {
		cfg.Tls.Key = *key
	}
	if cfg.Ca.Cert == "" || explicitFlags["ca"] {
		cfg.Ca.Cert = *caCert
	}
	if cfg.Ca.Addr == "" || explicitFlags["ca-addr"] {
		cfg.Ca.Addr = *caAddr
	}
}

func main() {
	flag.Parse()

	var cfg *settings.ChatCfg
	var err error

	if *cfgPath != "" {
		cfg, err = readCfgFile(*cfgPath)
		if err != nil {
			log.Fatal(err)
		}
		applyCmdArgs(cfg)
	} else {
		cfg = &settings.ChatCfg{
			Addr: *addr,
			Logs: *logDir,
			Tls: settings.TlsCfg{
				Cert: *cert,
				Key:  *key,
			},
			Ca: settings.CaCfg{
				Addr: *caAddr,
				Cert: *caCert,
			},
		}
	}

	authRepo, chatRepo := server.SetupPgxDb(context.Background())
	srv, err := server.SetupServer(cfg, authRepo, chatRepo)

	log := logging.GetLogger()

	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		srv.Close()
		log.Println("Closed server")
		os.Exit(0)
	}()

	fmt.Println("Server started on https://" + *addr)

	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
