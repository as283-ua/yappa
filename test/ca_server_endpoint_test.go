package test

import (
	"bytes"
	"crypto/rand"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/ca"
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

var DefaultCaArgs = &ca.CmdArgs{
	Addr:           "127.0.0.1:4436",
	Cert:           "../certs/ca_server/ca_server.crt",
	Key:            "../certs/ca_server/ca_server.key",
	ChatServerCert: "../certs/server/server.crt",
	RootCa:         "../certs/ca/ca.crt",
	CaKey:          "../certs/ca/ca.key",
}

func TestAllowNoCert(t *testing.T) {
	RunCaServer()

	client := GetHttp3Client("../certs", "", "../certs/ca/ca.crt")

	token := make([]byte, 64)
	rand.Read(token)

	allowUser := &gen.AllowUser{
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
	RunCaServer()
	client := GetHttp3Client("../certs", "server", "../certs/ca/ca.crt")

	token := make([]byte, 64)
	rand.Read(token)

	allowUser := &gen.AllowUser{
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
	RunCaServer()

	client := GetHttp3Client("../certs", "test", "../certs/ca/ca.crt")
	token := make([]byte, 64)
	rand.Read(token)

	allowUser := &gen.AllowUser{
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

func TestAllowAndSignCert(t *testing.T) {
	RunCaServer()

	clientServe := GetHttp3Client("../certs", "server", "../certs/ca/ca.crt")
	token := make([]byte, 64)
	rand.Read(token)

	allowUser := &gen.AllowUser{
		User:  "User1",
		Token: token,
	}

	data, err := proto.Marshal(allowUser)

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	resp, err := clientServe.Post("https://"+DefaultCaArgs.Addr+"/allow", "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("Status should be OK 200")
		t.FailNow()
	}

	clientUser := GetHttp3Client("../certs", "test", "../certs/ca/ca.crt")

	csr, err := os.ReadFile("../certs/test.csr")

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	certRequest := &gen.CertRequest{
		User:  "User1",
		Token: token,
		Csr:   csr,
	}

	data, _ = proto.Marshal(certRequest)

	resp, err = clientUser.Post("https://"+DefaultCaArgs.Addr+"/sign", "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK but got %v", resp.StatusCode)
		t.FailNow()
	}

	bytesResp, _ := io.ReadAll(resp.Body)

	certResponse := &gen.CertResponse{}
	proto.Unmarshal(bytesResp, certResponse)

	t.Log(string(certResponse.Cert))
}
