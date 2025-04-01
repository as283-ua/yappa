package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/client/settings"
	"google.golang.org/protobuf/proto"
)

type RegistrationClient struct {
	Client *http.Client
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
		return nil, err
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
