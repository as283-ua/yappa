package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/as283-ua/yappa/internal/server/handler"
	"github.com/as283-ua/yappa/internal/server/settings"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quic-go/quic-go/http3"
)

var (
	tlsConfig *tls.Config
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func setupDB(ctx context.Context) *pgxpool.Pool {
	user := getEnv("YAPPA_DB_USER", "yappa")
	host := getEnv("YAPPA_DB_HOST", "localhost:5432")
	pass, exists := os.LookupEnv("YAPPA_MASTER_KEY")

	if !exists {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("YAPPA_MASTER_KEY not set. Enter the password: ")
		password, _, err := reader.ReadLine()

		if err != nil {
			log.Fatalf("Error reading from stdin: %v", err)
		}

		pass = string(password)
	}

	uri := fmt.Sprintf("postgres://%v:%v@%v/yappa-chat", user, pass, host)

	pool, err := pgxpool.New(ctx, uri)
	if err != nil {
		log.Fatalf("Failed to create DB pool: %v", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}

	return pool
}

func getTlsConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(settings.ChatSettings.Cert, settings.ChatSettings.Key)
	if err != nil {
		return nil, err
	}

	rootCAs := x509.NewCertPool()
	caCertPath := settings.ChatSettings.CaCert

	caCertBytes, err := os.ReadFile(caCertPath)

	if err != nil {
		return nil, err
	}

	rootCAs.AppendCertsFromPEM(caCertBytes)

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		NextProtos:   []string{"h3"},
	}, nil
}

func SetupServer(cfg *settings.Settings) (*http3.Server, error) {
	settings.ChatSettings = cfg
	err := settings.ChatSettings.Validate()

	if err != nil {
		return nil, err
	}

	db.Pool = setupDB(context.Background())

	err = common.InitHttp3Client(settings.ChatSettings.CaCert)
	if err != nil {
		return nil, err
	}

	tlsConfig, err = getTlsConfig()

	if err != nil {
		return nil, err
	}

	router := http.NewServeMux()

	router.Handle("CONNECT /connect", http.HandlerFunc(handler.Connection))
	router.Handle("POST /register", http.HandlerFunc(handler.RegisterInit))

	server := &http3.Server{
		Addr:      settings.ChatSettings.Addr,
		Handler:   router,
		TLSConfig: tlsConfig,
	}

	return server, nil
}
