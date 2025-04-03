package settings

import "errors"

type ChatCfg struct {
	Addr   string
	Cert   string
	Key    string
	CaCert string
	CaAddr string
	LogDir string
}

func (c *ChatCfg) Validate() error {
	if c.Addr == "" {
		return errors.New("address must not be empty")
	}

	if c.Cert == "" {
		return errors.New("cert must not be empty")
	}

	if c.Key == "" {
		return errors.New("key must not be empty")
	}

	if c.CaCert == "" {
		return errors.New("ca certificate must not be empty")
	}

	if c.CaAddr == "" {
		return errors.New("ca host address must not be empty")
	}

	return nil
}

var (
	ChatSettings *ChatCfg
)
