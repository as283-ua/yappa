package test

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"google.golang.org/protobuf/proto"
)

func getHttp3Client(certificateOwner string) *http.Client {
	rootCAs := x509.NewCertPool()
	caCertPath := "../certs/ca/ca.crt"

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatal("Failed to read root CA certificate:", err)
	}

	rootCAs.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		RootCAs:    rootCAs,
		NextProtos: []string{"h3"},
	}

	if certificateOwner != "" {
		cert, err := tls.LoadX509KeyPair("../certs/"+certificateOwner+"/"+certificateOwner+".crt", "../certs/"+certificateOwner+"/"+certificateOwner+".key")
		if err != nil {
			log.Fatal(err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig:      &quic.Config{},
	}

	return &http.Client{
		Transport: transport,
	}
}

func TestAllowNoCert(t *testing.T) {
	client := getHttp3Client("")

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

	_, err = client.Post("https://yappa.io:4434/allow", "application/x-protobuf", bytes.NewReader(data))

	if err == nil {
		t.Error("Not providing a certificate should give an error")
	}
}

func TestAllowServerCert(t *testing.T) {
	client := getHttp3Client("server")

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

	resp, err := client.Post("https://yappa.io:4434/allow", "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("Status should be OK 200")
	}
}

func TestAllowTestCert(t *testing.T) {
	client := getHttp3Client("test")
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

	resp, err := client.Post("https://yappa.io:4434/allow", "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("Status should be unauthorized 401")
	}
}

func TestAllowAndSignCert(t *testing.T) {
	clientServe := getHttp3Client("server")
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

	resp, err := clientServe.Post("https://yappa.io:4434/allow", "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("Status should be OK 200")
		t.FailNow()
	}

	clientUser := getHttp3Client("test")

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

	resp, err = clientUser.Post("https://yappa.io:4434/sign", "application/x-protobuf", bytes.NewReader(data))

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
