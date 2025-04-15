package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/as283-ua/yappa/internal/server/auth"
	"github.com/as283-ua/yappa/internal/server/chat"
	"github.com/as283-ua/yappa/internal/server/connection"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/as283-ua/yappa/internal/server/settings"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/qlog"
	"golang.org/x/term"
)

var (
	tlsConfig     *tls.Config
	tlsVerifyOpts x509.VerifyOptions
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func SetupPgxDb(ctx context.Context) (*auth.PgxUserRepo, *chat.PgxChatRepo) {
	user := getEnv("YAPPA_DB_USER", "yappa")
	host := getEnv("YAPPA_DB_HOST", "localhost:5432")
	pass, exists := os.LookupEnv("YAPPA_MASTER_KEY")

	if !exists {
		fmt.Print("YAPPA_MASTER_KEY not set. Enter the password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
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

	return &auth.PgxUserRepo{Pool: pool}, &chat.PgxChatRepo{Pool: pool, Ctx: context.Background()}
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

	tlsVerifyOpts = x509.VerifyOptions{
		Roots: rootCAs,
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequestClientCert,
		NextProtos:   []string{"h3"},
		RootCAs:      rootCAs,
	}, nil
}

func SetupServer(cfg *settings.ChatCfg, userRepo auth.UserRepo, chatRepo chat.ChatRepo) (*http3.Server, error) {
	settings.ChatSettings = cfg
	err := settings.ChatSettings.Validate()

	if err != nil {
		return nil, err
	}

	auth.Repo = userRepo
	chat.Repo = chatRepo

	err = common.InitHttp3Client(settings.ChatSettings.CaCert)
	if err != nil {
		return nil, err
	}

	common.AddTlsCert(cfg.Cert, cfg.Key)

	tlsConfig, err = getTlsConfig()

	if err != nil {
		return nil, err
	}

	if cfg.LogDir != "" {
		err = logging.SetOutput(cfg.LogDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	router := http.NewServeMux()

	router.Handle("CONNECT /connect", connection.RequireCertificate(tlsVerifyOpts, http.HandlerFunc(connection.Connection)))
	router.Handle("GET /test", connection.RequireCertificate(tlsVerifyOpts, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte("good")) })))
	router.Handle("POST /register", http.HandlerFunc(auth.RegisterInit))
	router.Handle("POST /register/confirm", http.HandlerFunc(auth.RegisterComplete))

	server := &http3.Server{
		Addr:        settings.ChatSettings.Addr,
		Handler:     router,
		TLSConfig:   tlsConfig,
		IdleTimeout: 60 * time.Second,
		QUICConfig: &quic.Config{
			EnableDatagrams: true,
			Tracer:          qlog.DefaultConnectionTracer,
		},
	}

	server.EnableDatagrams = true

	return server, nil
}
