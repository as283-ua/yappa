package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
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

	if !strings.HasSuffix(*certDir, "/") {
		*certDir += "/"
	}

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

	err := service.InitHttp3Client(*caCert)
	if err != nil {
		log.Fatalf("Failed to create http client: %v", err)
	}

	_, err = os.Stat(settings.CliSettings.CertDir + "yappa.crt")
	if err == nil {
		_, err = os.Stat(settings.CliSettings.CertDir + "yappa.key")
		if err == nil {
			err = service.UseCertificate(settings.CliSettings.CertDir+"yappa.crt",
				settings.CliSettings.CertDir+"yappa.key")
			if err != nil {
				log.Fatalf("Failed to add certificate to http client: %v", err)
			}

			err = service.UseMlkemKey(settings.CliSettings.CertDir + "dk.key")
			if err != nil {
				log.Fatalf("Failed getting private mlkem key: %v", err)
			}

			h3c, _ := service.GetHttp3Client()
			chatClient := service.InitChatClient(h3c)
			go func() {
				err = chatClient.Connect()
				if err != nil {
					log.Printf("Failed opening connection to the server: %v", err)
					return
				}
			}()
			defer service.GetChatClient().Close()
		} else {
			log.Fatal("Certificate found but missing private key")
		}
	} else {
		log.Println("No certificate found. Must register")
	}

	saveState, err := save.LoadChats()
	if err != nil {
		log.Fatalf("Failed to load saved chats: %v", err)
	}

	go func() {
		chatCli := service.GetChatClient()
		chatMap := make(map[[32]byte]*client.Chat)
		<-chatCli.ConnectedC
		for chatCli.GetConnected() {
			msg := <-chatCli.MainSub
			switch payload := msg.Payload.(type) {
			case *server.ServerMessage_Send:
				chat, ok := chatMap[[32]byte(payload.Send.InboxId)]
				if !ok {
					chat = save.DirectChat(saveState, payload.Send.InboxId)
					chatMap[[32]byte(payload.Send.InboxId)] = chat
				}
				event, currentSerial, key, err := service.DecryptPeerMessage(chat, payload)
				if err != nil {
					log.Println("Error decrypting peer msg:", err)
					break
				}
				save.NewEvent(chat, currentSerial, key, event)
			}
		}
	}()
	defer func() {
		save.SaveChats(saveState)
		log.Println("Saved chats state")
	}()

	if service.GetUsername() != "" {
		newChats, err := service.GetChatClient().GetNewChats(saveState)
		if err != nil {
			log.Printf("Errors while retrieving new chats: %v", err)
		}
		for _, chat := range newChats {
			save.NewDirectChat(saveState, chat)
		}

		newMsgs, err := service.GetChatClient().GetNewMessages(saveState)
		if err != nil {
			log.Printf("Errors while retrieving new messages: %v", err)
		}
		for chat, events := range newMsgs {
			for _, ev := range events {
				save.NewEvent(chat, ev)
			}
		}
	}

	log.Println("Started client")
	p := tea.NewProgram(ui.NewMainPage(saveState))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
