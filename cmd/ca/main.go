package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/as283-ua/yappa/internal/ca"
	"github.com/as283-ua/yappa/internal/ca/logging"
	"github.com/as283-ua/yappa/internal/ca/settings"
)

var (
	addr           = flag.String("addr", "0.0.0.0:4434", "Host IP and port")
	cert           = flag.String("cert", "certs/ca_tls/ca_tls.crt", "TLS Certificate")
	key            = flag.String("key", "certs/ca_tls/ca_tls.key", "TLS Key")
	chatServerCert = flag.String("server-cert", "certs/server/server.crt", "TLS Certificate for chat server")
	rootCa         = flag.String("ca", "certs/ca/ca.crt", "Root CA certificate")
	caKey          = flag.String("ca-key", "certs/ca/ca.key", "Root CA private key")
	logDir         = flag.String("logs", "logs/ca/", "Log directory")
	cfgPath        = flag.String("config", "cfg/yappacad.toml", "Configuration file")
)

func readCfgFile(path string) (*settings.CaCfg, error) {
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

	cfg := &settings.CaCfg{}
	_, err = toml.Decode(string(cfgRaw), &cfg)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// apply default values or those specified explicitly through command line options
func applyCmdArgs(cfg *settings.CaCfg) {
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
	if cfg.Cacert == "" || explicitFlags["cacert"] {
		cfg.Cacert = *rootCa
	}
	if cfg.Key == "" || explicitFlags["ca-key"] {
		cfg.Key = *caKey
	}
	if cfg.Chat.Cert == "" || explicitFlags["server-cert"] {
		cfg.Chat.Cert = *chatServerCert
	}
}

func main() {
	flag.Parse()
	var cfg *settings.CaCfg
	var err error

	if *cfgPath != "" {
		cfg, err = readCfgFile(*cfgPath)
		if err != nil {
			log.Fatal(err)
		}
		applyCmdArgs(cfg)
	} else {
		cfg = &settings.CaCfg{
			Addr: *addr,
			Tls: settings.TlsCfg{
				Cert: *cert,
				Key:  *key,
			},
			Chat: settings.ChatServerCfg{
				Cert: *chatServerCert,
			},
			Cacert: *rootCa,
			Key:    *caKey,
			Logs:   *logDir,
		}
	}

	server, err := ca.SetupServer(cfg)

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
		log.Fatal("Error setting up server:", err)
	}

	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")

	fmt.Println("CA Server started on https://" + *addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
