package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/as283-ua/yappa/internal/client/save"
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
	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")

	certDir = flag.String("certs", "certs/client/", "Path to certs directory")
	caCert = flag.String("cacert", "certs/ca/ca.crt", "Yappa CA certificate")
	serverHost = flag.String("server", "yappa.io:4433", "Yappa chat server ip and port")
	caHost = flag.String("ca", "yappa.io:4434", "Yappa CA server ip and port")
	logsDir = flag.String("logs", "logs/cli/", "Error logs directory.\n\"/dev/null\" or \"null\" to suppress error logs.\n\"-\" to show errors on-screen (buggy)")

	flag.Parse()

	settings.CliSettings = settings.Settings{
		CertDir:    *certDir,
		CaCert:     *caCert,
		ServerHost: *serverHost,
		CaHost:     *caHost,
	}

	var logFile *os.File = nil
	if *logsDir == "/dev/null" || *logsDir == "null" {
		logFile, _ = os.Open(os.DevNull)
	} else if *logsDir != "-" {
		filename := time.Now().Format("2006-01-02_15-04-05") + "-session.log"

		_, err := os.Stat(*logsDir)
		if err != nil {
			os.Mkdir(*logsDir, 0755)
		}

		logFile, err = os.OpenFile(*logsDir+filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer logFile.Close()
	}

	if logFile != nil {
		log.SetOutput(logFile)
	}

	service.InitHttp3Client(*caCert)

	saveState, err := save.LoadChats()
	if err != nil {
		log.Fatalf("Failed to load saved chats: %v", err)
	}
	defer save.SaveChats(saveState)

	p := tea.NewProgram(ui.NewMainPage(saveState))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

}
