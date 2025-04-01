package settings

type Settings struct {
	CertDir    string
	CaCert     string
	ServerHost string
	CaHost     string
}

var CliSettings Settings
