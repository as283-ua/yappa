package settings

import "fmt"

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
		return fmt.Errorf("address must not be empty")
	}

	if c.Cert == "" {
		return fmt.Errorf("cert must not be empty")
	}

	if c.Key == "" {
		return fmt.Errorf("key must not be empty")
	}

	if c.CaCert == "" {
		return fmt.Errorf("ca certificate must not be empty")
	}

	if c.CaAddr == "" {
		return fmt.Errorf("ca host address must not be empty")
	}
	return nil
}

var (
	ChatSettings *ChatCfg
)
