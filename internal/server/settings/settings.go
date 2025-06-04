package settings

import "errors"

// type ChatCfg struct {
// 	Addr   string
// 	Cert   string
// 	Key    string
// 	CaCert string
// 	CaAddr string
// 	LogDir string
// }

type ChatCfg struct {
	Addr string
	Logs string
	Tls  TlsCfg `toml:"tls"`
	Ca   CaCfg  `toml:"ca"`
}

type TlsCfg struct {
	Cert string
	Key  string
}

type CaCfg struct {
	Addr string
	Cert string
}

func (c *ChatCfg) Validate() error {
	if c.Addr == "" {
		return errors.New("address must not be empty")
	}

	if c.Tls.Cert == "" {
		return errors.New("cert must not be empty")
	}

	if c.Tls.Key == "" {
		return errors.New("key must not be empty")
	}

	if c.Ca.Cert == "" {
		return errors.New("ca certificate must not be empty")
	}

	if c.Ca.Addr == "" {
		return errors.New("ca host address must not be empty")
	}

	return nil
}

var (
	ChatSettings *ChatCfg
)
