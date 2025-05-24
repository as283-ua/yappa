package test

import (
	"bytes"
	"crypto/rand"
	"log"
	"net/http"
	"testing"

	ca_proto "github.com/as283-ua/yappa/api/gen/ca"
	"github.com/as283-ua/yappa/internal/ca"
	"github.com/as283-ua/yappa/internal/ca/settings"
	"github.com/quic-go/quic-go/http3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func RunCaServer() *http3.Server {
	server, err := ca.SetupServer(DefaultCaArgs)

	if err != nil {
		log.Fatal("Error booting server: ", err)
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	return server
}

var DefaultCaArgs = &settings.CaCfg{
	Addr:           "127.0.0.1:4436",
	Cert:           "../certs/ca_server/ca_server.crt",
	Key:            "../certs/ca_server/ca_server.key",
	ChatServerCert: "../certs/server/server.crt",
	RootCa:         "../certs/ca/ca.crt",
	CaKey:          "../certs/ca/ca.key",
}

func TestAllowNoCert(t *testing.T) {
	setup()

	client := GetHttp3Client("../certs", "", "../certs/ca/ca.crt")

	token := make([]byte, 64)
	rand.Read(token)

	allowUser := &ca_proto.AllowUser{
		User:  "User1",
		Token: token,
	}

	data, err := proto.Marshal(allowUser)

	assert.NoError(t, err)

	resp, err := client.Post("https://"+DefaultCaArgs.Addr+"/allow", "application/x-protobuf", bytes.NewReader(data))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Status should be", http.StatusUnauthorized)
}

func TestAllowServerCert(t *testing.T) {
	setup()
	client := GetHttp3Client("../certs", "server", "../certs/ca/ca.crt")

	token := make([]byte, 64)
	rand.Read(token)

	allowUser := &ca_proto.AllowUser{
		User:  "User1",
		Token: token,
	}

	data, err := proto.Marshal(allowUser)

	assert.NoError(t, err)

	resp, err := client.Post("https://"+DefaultCaArgs.Addr+"/allow", "application/x-protobuf", bytes.NewReader(data))

	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status should be", http.StatusOK)
}

func TestAllowTestCert(t *testing.T) {
	setup()

	client := GetHttp3Client("assets", "test_ok", "../certs/ca/ca.crt")
	token := make([]byte, 64)
	rand.Read(token)

	allowUser := &ca_proto.AllowUser{
		User:  "User1",
		Token: token,
	}

	data, err := proto.Marshal(allowUser)

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	resp, err := client.Post("https://"+DefaultCaArgs.Addr+"/allow", "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("Status should be unauthorized 401")
	}
}
