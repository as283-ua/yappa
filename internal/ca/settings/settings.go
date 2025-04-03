package settings

import "fmt"

type CaCfg struct {
	Addr           string
	Cert           string
	Key            string
	ChatServerCert string
	RootCa         string
	CaKey          string
}

func (c *CaCfg) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address must not be empty")
	}

	if c.Cert == "" {
		return fmt.Errorf("cert must not be empty")
	}

	if c.Key == "" {
		return fmt.Errorf("key must not be empty")
	}

	if c.ChatServerCert == "" {
		return fmt.Errorf("chatServerCert must not be empty")
	}

	if c.RootCa == "" {
		return fmt.Errorf("rootCa must not be empty")
	}

	if c.CaKey == "" {
		return fmt.Errorf("caKey must not be empty")
	}

	return nil
}

var (
	CaSettings *CaCfg
)
