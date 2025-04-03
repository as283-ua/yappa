package settings

import "errors"

type CaCfg struct {
	Addr           string
	Cert           string
	Key            string
	ChatServerCert string
	RootCa         string
	CaKey          string
	LogDir         string
}

func (c *CaCfg) Validate() error {
	if c.Addr == "" {
		return errors.New("address must not be empty")
	}

	if c.Cert == "" {
		return errors.New("cert must not be empty")
	}

	if c.Key == "" {
		return errors.New("key must not be empty")
	}

	if c.ChatServerCert == "" {
		return errors.New("chatServerCert must not be empty")
	}

	if c.RootCa == "" {
		return errors.New("rootCa must not be empty")
	}

	if c.CaKey == "" {
		return errors.New("caKey must not be empty")
	}

	return nil
}

var (
	CaSettings *CaCfg
)
