package test

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/client/service"
	"github.com/as283-ua/yappa/internal/server"
	"github.com/as283-ua/yappa/internal/server/auth"
	"github.com/as283-ua/yappa/internal/server/chat"
	"github.com/as283-ua/yappa/internal/server/settings"
	"github.com/as283-ua/yappa/test/mock"
	"github.com/quic-go/quic-go/http3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var caServer, chatServer *http3.Server

func setup() {
	os.Setenv("YAPPA_MASTER_KEY", "pass")

	if caServer == nil {
		caServer = RunCaServer()
	}

	if chatServer == nil {
		chatServer = RunChatServer()
	}
}

func RunChatServer() *http3.Server {
	server, err := server.SetupServer(&DefaultChatServerArgs,
		mock.EmptyMockUserRepo(),
		mock.EmptyMockChatRepo())

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

var DefaultChatServerArgs settings.ChatCfg = settings.ChatCfg{
	Addr:   "127.0.0.1:4435",
	Cert:   "../certs/server/server.crt",
	Key:    "../certs/server/server.key",
	CaCert: "../certs/ca/ca.crt",
	CaAddr: DefaultCaArgs.Addr,
}

func TestRegister(t *testing.T) {
	setup()

	client := GetHttp3Client("../certs", "", DefaultChatServerArgs.CaCert)

	username := "User1"

	regRequest := &gen.RegistrationRequest{
		User: username,
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

	certBundle, err := service.GeneratePrivKey()
	assert.NoError(t, err)

	csr, err := service.GenerateCSR(certBundle.Key, username)

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

func TestRequireCertClient(t *testing.T) {
	setup()

	t.Run("no_cert_errors", func(t *testing.T) {
		client := GetHttp3Client("../certs", "", DefaultChatServerArgs.CaCert)

		r, err := client.Get("https://" + DefaultChatServerArgs.Addr + "/test")
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, r.StatusCode, http.StatusBadRequest)
	})

	t.Run("with_cert_ok", func(t *testing.T) {
		client := GetHttp3Client("../certs", "test_ok", DefaultChatServerArgs.CaCert)

		r, err := client.Get("https://" + DefaultChatServerArgs.Addr + "/test")
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, r.StatusCode, http.StatusOK)
	})

	t.Run("with_incorrect_cert_errors", func(t *testing.T) {
		client := GetHttp3Client("../certs", "test_bad", DefaultChatServerArgs.CaCert)

		r, err := client.Get("https://" + DefaultChatServerArgs.Addr + "/test")
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, r.StatusCode, http.StatusBadRequest)
	})
}

func TestConnection(t *testing.T) {
	setup()

	t.Run("send_init_chat_type", func(t *testing.T) {
		t.Skip()

		// not very reliable test, just proof of concept
		serverURL := "https://" + DefaultChatServerArgs.Addr + "/connect"
		u, err := url.Parse(serverURL)
		if !assert.NoError(t, err) {
			return
		}
		client := GetHttp3Client("../certs", "test_ok", DefaultChatServerArgs.CaCert)
		ecdhKX25519, err := ecdh.X25519().GenerateKey(rand.Reader)
		if !assert.NoError(t, err) {
			return
		}
		ecdhBytes := ecdhKX25519.PublicKey().Bytes()
		ecdhStr := base64.StdEncoding.EncodeToString(ecdhBytes)
		header := http.Header{}
		header.Add("X-Ecdh", ecdhStr)
		str, err := Http3Stream(context.Background(), u, client.Transport.(*http3.Transport), header)
		if !assert.NoError(t, err) {
			return
		}

		defer str.Close()

		msg := &gen.ClientMessage{
			Payload: &gen.ClientMessage_Init{Init: &gen.ChatInit{
				InboxId: []byte{0, 1, 2, 3},
			}},
		}

		m, err := proto.Marshal(msg)
		if !assert.NoError(t, err) {
			return
		}

		// length of message at the start of the frame
		messageLen := len(m)
		lenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBytes, uint32(messageLen))

		str.Write(append(lenBytes, m...))
		// timer := time.NewTimer(5 * time.Minute)
		// <-timer.C
	})

	t.Run("require_ecdh_no_ecdh", func(t *testing.T) {
		serverURL := "https://" + DefaultChatServerArgs.Addr + "/connect"
		u, err := url.Parse(serverURL)
		if !assert.NoError(t, err) {
			return
		}
		client := GetHttp3Client("../certs", "test_ok", DefaultChatServerArgs.CaCert)
		_, err = Http3Stream(context.Background(), u, client.Transport.(*http3.Transport), http.Header{})
		if assert.Error(t, err) {
			t.Log(err)
			return
		}
	})

	t.Run("require_ecdh_bad_ecdh", func(t *testing.T) {
		serverURL := "https://" + DefaultChatServerArgs.Addr + "/connect"
		u, err := url.Parse(serverURL)
		if !assert.NoError(t, err) {
			return
		}
		client := GetHttp3Client("../certs", "test_ok", DefaultChatServerArgs.CaCert)
		ecdhK256, err := ecdh.P256().GenerateKey(rand.Reader)
		if !assert.NoError(t, err) {
			return
		}
		ecdhBytes := ecdhK256.PublicKey().Bytes()
		ecdhStr := base64.StdEncoding.EncodeToString(ecdhBytes)
		header := http.Header{}
		header.Add("X-Ecdh", ecdhStr)
		_, err = Http3Stream(context.Background(), u, client.Transport.(*http3.Transport), header)
		if assert.Error(t, err) {
			t.Log(err)
			return
		}
	})

	t.Run("require_ecdh_success", func(t *testing.T) {
		serverURL := "https://" + DefaultChatServerArgs.Addr + "/connect"
		u, err := url.Parse(serverURL)
		if !assert.NoError(t, err) {
			return
		}
		client := GetHttp3Client("../certs", "test_ok", DefaultChatServerArgs.CaCert)
		ecdhKX25519, err := ecdh.X25519().GenerateKey(rand.Reader)
		if !assert.NoError(t, err) {
			return
		}
		ecdhBytes := ecdhKX25519.PublicKey().Bytes()
		ecdhStr := base64.StdEncoding.EncodeToString(ecdhBytes)
		header := http.Header{}
		header.Add("X-Ecdh", ecdhStr)
		_, err = Http3Stream(context.Background(), u, client.Transport.(*http3.Transport), header)
		if !assert.NoError(t, err) {
			t.Log(err)
			return
		}
	})
}

func TestChatInit(t *testing.T) {
	setup()

	t.Run("init_chat", func(t *testing.T) {
		client := GetHttp3Client("../certs", "test_ok", DefaultChatServerArgs.CaCert)

		inboxId := make([]byte, 32)
		rand.Read(inboxId)
		regRequest := &gen.ChatInit{
			InboxId: inboxId,
		}

		data, err := proto.Marshal(regRequest)

		assert.NoError(t, err)
		url := fmt.Sprintf("https://%v/chat/init", DefaultChatServerArgs.Addr)
		resp, err := client.Post(url, "application/x-protobuf", bytes.NewReader(data))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer resp.Body.Close()
		assert.Equal(t, resp.StatusCode, http.StatusOK)
		repo, _ := chat.Repo.(*mock.MockChatRepo)
		assert.Equal(t, len(repo.GetChatInboxes()), 1)
		assert.Equal(t, repo.GetChatInboxes()[0].Code, inboxId)
	})

	t.Run("notify_chat", func(t *testing.T) {
		client := GetHttp3Client("../certs", "test_ok", DefaultChatServerArgs.CaCert)

		inboxId := make([]byte, 32)
		rand.Read(inboxId)
		regRequest := &gen.ChatInit{
			InboxId: inboxId,
		}

		data, err := proto.Marshal(regRequest)

		assert.NoError(t, err)
		url := fmt.Sprintf("https://%v/chat/init", DefaultChatServerArgs.Addr)
		resp, err := client.Post(url, "application/x-protobuf", bytes.NewReader(data))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer resp.Body.Close()
		ecdhReceiver, _ := ecdh.X25519().GenerateKey(rand.Reader)
		ecdhSender, _ := ecdh.X25519().GenerateKey(rand.Reader)

		receiverUsername := "Receiver"
		auth.Repo.CreateUser(context.Background(), receiverUsername, "")
		auth.Repo.ChangeEcdhTemp(context.Background(), receiverUsername, ecdhReceiver.PublicKey().Bytes())
		aesK, _ := ecdhSender.ECDH(ecdhReceiver.PublicKey())

		encSender, _ := Encrypt([]byte("Sender"), aesK)
		encInboxId, _ := Encrypt(inboxId, aesK)

		regRequest2 := &gen.ChatInitNotify{
			Receiver:   receiverUsername,
			EcdhPub:    ecdhSender.PublicKey().Bytes(),
			EncSender:  encSender,
			EncInboxId: encInboxId,
		}

		data, _ = proto.Marshal(regRequest2)
		url = fmt.Sprintf("https://%v/chat/notify", DefaultChatServerArgs.Addr)
		resp2, err := client.Post(url, "application/x-protobuf", bytes.NewReader(data))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer resp2.Body.Close()

		newChats, err := chat.Repo.GetNewChats(receiverUsername)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, 1, len(newChats))
		ecdhPub, err := ecdh.X25519().NewPublicKey(newChats[0].EcdhPub)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		aesK2, _ := ecdhReceiver.ECDH(ecdhPub)

		inboxIdDec, err := Decrypt(newChats[0].EncInboxCode, aesK2)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, inboxId, inboxIdDec)

		senderDec, err := Decrypt(newChats[0].EncSender, aesK2)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, "Sender", string(senderDec))
	})
}
