package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/as283-ua/yappa/internal/client/service"
	"github.com/as283-ua/yappa/internal/client/settings"
	"github.com/as283-ua/yappa/internal/client/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	certDir    *string
	caCert     *string
	serverHost *string
	caHost     *string
	logsDir    *string
)

func main() {
	certDir = flag.String("certs", "certs/client", "Path to certs directory")
	caCert = flag.String("cacert", "certs/ca/ca.crt", "Yappa CA certificate")
	serverHost = flag.String("server", "yappa.io:4433", "Yappa chat server ip and port")
	caHost = flag.String("ca", "yappa.io:4434", "Yappa CA server ip and port")
	logsDir = flag.String("logs", "logs/", "Error logs directory. \"/dev/null\" or \"null\" to suppress error logs")

	flag.Parse()

	settings.CliSettings = settings.Settings{
		CertDir:    *certDir,
		CaCert:     *caCert,
		ServerHost: *serverHost,
		CaHost:     *caHost,
	}

	var logFile *os.File
	if *logsDir == "/dev/null" || *logsDir == "null" {
		logFile, _ = os.Open(os.DevNull)
	} else {
		filename := time.Now().Format("2006-01-02_15-04-05") + "-yappa-err.log"

		_, err := os.Stat(*logsDir)
		if err != nil {
			os.Mkdir(*logsDir, 0755)
		}

		logFile, err = os.OpenFile(*logsDir+filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer logFile.Close()
	}

	log.SetOutput(logFile)

	service.InitHttp3Client(*caCert)

	p := tea.NewProgram(ui.NewMainPage())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
