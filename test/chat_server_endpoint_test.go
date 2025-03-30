package test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/server"
	"github.com/as283-ua/yappa/internal/server/settings"
	"github.com/quic-go/quic-go/http3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func RunChatServer() *http3.Server {
	server, err := server.SetupServer(&DefaultChatServerArgs)

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

var DefaultChatServerArgs settings.Settings = settings.Settings{
	Addr:   "127.0.0.1:4435",
	Cert:   "../certs/server/server.crt",
	Key:    "../certs/server/server.key",
	CaCert: "../certs/ca/ca.crt",
	CaAddr: DefaultCaArgs.Addr,
}

func TestRegister(t *testing.T) {
	os.Setenv("YAPPA_MASTER_KEY", "pass")

	RunCaServer()
	RunChatServer()

	client := GetHttp3Client("../certs", "", DefaultChatServerArgs.CaCert)

	regRequest := &gen.RegistrationRequest{
		User: "User1",
	}

	data, err := proto.Marshal(regRequest)

	assert.NoError(t, err)
	url := fmt.Sprintf("https://%v/register", DefaultChatServerArgs.Addr)
	resp, err := client.Post(url, "application/x-protobuf", bytes.NewReader(data))

	assert.NoError(t, err)

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	allowUser := &gen.AllowUser{}
	err = proto.Unmarshal(body, allowUser)

	assert.NoError(t, err)

	assert.Equal(t, regRequest.User, allowUser.User)

	csr, err := os.ReadFile("../certs/test.csr")

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	certRequest := &gen.CertRequest{
		User:  allowUser.User,
		Token: allowUser.Token,
		Csr:   csr,
	}

	data, _ = proto.Marshal(certRequest)

	resp, err = client.Post("https://"+DefaultCaArgs.Addr+"/sign", "application/x-protobuf", bytes.NewReader(data))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bytesResp, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	certResponse := &gen.CertResponse{}
	proto.Unmarshal(bytesResp, certResponse)

	t.Log(string(certResponse.Cert))

	confirmation := &gen.ConfirmRegistration{
		User:  regRequest.User,
		Token: certResponse.Token,
		Cert:  certResponse.Cert,
	}

	data, _ = proto.Marshal(confirmation)

	resp, err = client.Post("https://"+DefaultChatServerArgs.Addr+"/register/confirm", "application/x-protobuf", bytes.NewReader(data))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()
}
