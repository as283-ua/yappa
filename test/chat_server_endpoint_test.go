package test

import (
	"log"
	"testing"

	"github.com/as283-ua/yappa/internal/server"
	"github.com/quic-go/quic-go/http3"
)

func runChatServer() *http3.Server {
	server, err := server.SetupServer(defaultChatServerArguments())

	if err != nil {
		log.Fatal("Error booting server: ", err)
	}

	go func() {
		server.ListenAndServe()
	}()

	return server
}

func defaultChatServerArguments() *server.CmdArgs {
	return &server.CmdArgs{
		Addr: "127.0.0.1:4435",
		Cert: "../certs/ca_server/ca_server.crt",
		Key:  "../certs/ca_server/ca_server.key",
	}
}

func TestRegister(t *testing.T) {
	server := runChatServer()
	defer server.Close()

}
