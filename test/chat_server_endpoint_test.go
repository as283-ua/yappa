package test

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
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
	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/as283-ua/yappa/internal/server/settings"
	"github.com/jackc/pgx/v5"
	"github.com/quic-go/quic-go/http3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type MockUserRepo struct {
	users  map[string]db.User
	serial int
}

func (r MockUserRepo) GetUserByUsername(ctx context.Context, user string) (db.User, error) {
	u, ok := r.users[user]
	if !ok {
		return u, pgx.ErrNoRows
	}
	return u, nil
}

func (r *MockUserRepo) CreateUser(ctx context.Context, user, cert string) error {
	_, err := r.GetUserByUsername(ctx, user)
	if err == nil {
		return errors.New("User already exists")
	}
	r.users[user] = db.User{ID: int32(r.serial), Username: user, Certificate: cert}
	r.serial++
	return nil
}

func (r *MockUserRepo) ChangeEcdhTemp(ctx context.Context, username string, ecdh []byte) error {
	user, err := r.GetUserByUsername(ctx, username)
	if err != nil {
		return errors.New("User doesn't exist")
	}

	user.EcdhTemp = ecdh
	r.users[username] = user
	return nil
}

type MockChatRepo struct {
}

func (r MockChatRepo) ShareChatInbox(username string, encSender, encInboxCode, ecdhPub []byte) error {
	return nil
}

func (r MockChatRepo) CreateChatInbox(inboxCode []byte) error {
	return nil
}

func (r MockChatRepo) GetNewChats(username string) ([]db.GetNewUserInboxesRow, error) {
	return nil, nil
}

func (r MockChatRepo) DeleteNewChats(username string) error {
	return nil
}

func (r MockChatRepo) SetInboxToken(inboxCode, token, encToken []byte) error {
	return nil
}

func (r MockChatRepo) GetToken(inboxCode []byte) ([]byte, error) {
	return nil, nil
}

func (r MockChatRepo) AddMessage(inboxCode, encMsg []byte) error {
	return nil
}

func (r MockChatRepo) GetMessages(inboxCode []byte) ([][]byte, error) {
	return nil, nil
}

func (r MockChatRepo) FlushInbox(inboxCode []byte) error {
	return nil
}

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
		&MockUserRepo{users: make(map[string]db.User), serial: 0},
		&MockChatRepo{})

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
