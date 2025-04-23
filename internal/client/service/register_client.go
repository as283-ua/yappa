package service

import (
	"bytes"
	"crypto/mlkem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/client/settings"
	"google.golang.org/protobuf/proto"
)

type RegistrationClient struct {
	Client *http.Client
}

func handleHttpErrors(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return errors.New("the server seems to be down")
		}

		log.Println("Network error:", netErr)
		return errors.New("network error")
	}

	if errors.Is(err, http.ErrServerClosed) {
		log.Println("Network error:", err)
		return errors.New("server closed the connection unexpectedly")
	}

	log.Println("Request failed:", err)
	return errors.New("request failed")
}

func (c RegistrationClient) RequestRegistration(username string) (*gen.AllowUser, error) {
	regRequest := &gen.RegistrationRequest{
		User: username,
	}

	data, err := proto.Marshal(regRequest)

	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%v/register", settings.CliSettings.ServerHost)

	resp, err := c.Client.Post(url, "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		return nil, handleHttpErrors(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if http.StatusOK != resp.StatusCode {
		return nil, fmt.Errorf("%v", string(body))
	}

	allowUser := &gen.AllowUser{}
	err = proto.Unmarshal(body, allowUser)

	if err != nil {
		return nil, err
	}

	return allowUser, nil
}

func (c RegistrationClient) CertificateSignatureRequest(allowUser *gen.AllowUser, csrPem []byte) (*gen.CertResponse, error) {
	certRequest := &gen.CertRequest{
		User:  allowUser.User,
		Token: allowUser.Token,
		Csr:   csrPem,
	}

	data, err := proto.Marshal(certRequest)
	if err != nil {
		log.Println("Protobuf marshal error:", err)
		return nil, errors.New("internal error")
	}

	resp, err := c.Client.Post("https://"+settings.CliSettings.CaHost+"/sign", "application/x-protobuf", bytes.NewReader(data))
	if err != nil {
		return nil, handleHttpErrors(err)
	}

	bytesResp, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("File read error:", err)
		return nil, errors.New("internal error")
	}
	defer resp.Body.Close()

	if http.StatusOK != resp.StatusCode {
		return nil, errors.New(string(bytesResp))
	}

	certResponse := &gen.CertResponse{}
	err = proto.Unmarshal(bytesResp, certResponse)
	if err != nil {
		log.Println("Protobuf unmarshal error:", err)
		return nil, errors.New("internal error")
	}

	return certResponse, nil
}

func (c RegistrationClient) CompleteRegistration(username string, certResponse *gen.CertResponse, kyberKey *mlkem.DecapsulationKey1024) error {
	confirmation := &gen.ConfirmRegistration{
		User:           username,
		Token:          certResponse.Token,
		Cert:           certResponse.Cert,
		PubKeyExchange: kyberKey.EncapsulationKey().Bytes(),
	}

	data, err := proto.Marshal(confirmation)
	if err != nil {
		log.Println("Protobuf marshal error:", err)
		return errors.New("internal error")
	}

	resp, err := c.Client.Post("https://"+settings.CliSettings.ServerHost+"/register/confirm", "application/x-protobuf", bytes.NewReader(data))

	if err != nil {
		return handleHttpErrors(err)
	}

	if http.StatusOK != resp.StatusCode {
		return errors.New("unexpected server response: " + resp.Status)
	}

	defer resp.Body.Close()
	return nil
}
