package settings

import "errors"

type CaCfg struct {
	Addr   string
	Logs   string
	Cacert string
	Key    string
	Tls    TlsCfg        `toml:"tls"`
	Chat   ChatServerCfg `toml:"chat"`
}

type TlsCfg struct {
	Cert string
	Key  string
}

type ChatServerCfg struct {
	Cert string
}

func (c *CaCfg) Validate() error {
	if c.Addr == "" {
		return errors.New("address must not be empty")
	}

	if c.Tls.Cert == "" {
		return errors.New("cert must not be empty")
	}

	if c.Tls.Key == "" {
		return errors.New("key must not be empty")
	}

	if c.Chat.Cert == "" {
		return errors.New("chatServerCert must not be empty")
	}

	if c.Cacert == "" {
		return errors.New("rootCa must not be empty")
	}

	if c.Key == "" {
		return errors.New("caKey must not be empty")
	}

	return nil
}

var (
	CaSettings *CaCfg
)
