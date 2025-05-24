package service

import (
	"fmt"
	"io"
	"net/http"

	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/settings"
	"google.golang.org/protobuf/proto"
)

type UsersClient struct {
	Client *http.Client
}

func (c UsersClient) GetUsers(page, size int, username string) ([]string, error) {
	url := fmt.Sprintf("https://%v/users", settings.CliSettings.ServerHost)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("page", fmt.Sprintf("%d", page))
	req.Header.Set("size", fmt.Sprintf("%d", size))
	req.Header.Set("name", username)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, handleHttpErrors(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", string(body))
	}

	var userResp server.Usernames
	err = proto.Unmarshal(body, &userResp)
	if err != nil {
		return nil, err
	}

	return userResp.Usernames, nil
}

func (c UsersClient) GetUserData(username string) (*server.UserData, error) {
	url := fmt.Sprintf("https://%v/users/%s", settings.CliSettings.ServerHost, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, handleHttpErrors(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user %q not found", username)
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v", string(body))
	}

	userData := &server.UserData{}
	err = proto.Unmarshal(body, userData)
	if err != nil {
		return nil, err
	}

	return userData, nil
}
