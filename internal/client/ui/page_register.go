package ui

import (
	"crypto/mlkem"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/as283-ua/yappa/api/gen/ca"
	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/internal/client/service"
	"github.com/as283-ua/yappa/internal/client/settings"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type RegisterOpt struct {
	username string
}

func (r RegisterOpt) String() string {
	return "Register"
}

func (r RegisterOpt) Select(_ *cli_proto.SaveState) (tea.Model, tea.Cmd) {
	return nil, register(r.username)
}

func register(username string) tea.Cmd {
	return func() tea.Msg {
		if len(username) < 4 {
			return fmt.Errorf("username must be at least 4 characters long: %v", username)
		}

		c, err := service.GetHttp3Client()
		if err != nil {
			return err
		}

		yc := service.RegistrationClient{Client: c}
		allowUser, err := yc.RequestRegistration(username)

		if err != nil {
			return err
		}
		return allowUser
	}
}

type RegistrationSuccess struct {
	Username string
}

type RegisterPage struct {
	username textinput.Model

	registerBtn *RegisterOpt
	options     []Option
	show        bool
	inputs      Inputs

	errorMessage string

	save *cli_proto.SaveState
}

func (m RegisterPage) GetOptions() []Option {
	return m.options
}

func (m RegisterPage) GetSelected() Option {
	return m.options[0]
}

func (m *RegisterPage) Up() {

}

func (m *RegisterPage) Down() {

}

func (m RegisterPage) GetInputs() Inputs {
	return m.inputs
}

func (m RegisterPage) ToggleShow() Inputer {
	m.show = !m.show
	return m
}

func (m RegisterPage) Save() *cli_proto.SaveState {
	return m.save
}

func NewRegisterPage(save *cli_proto.SaveState) RegisterPage {
	if save == nil {
		log.Println("nil save state")
		save = &cli_proto.SaveState{}
	}
	options := make([]Option, 0, 2)

	registerBtn := &RegisterOpt{username: ""}

	options = append(options, registerBtn)

	options = append(options, Exit{})

	inputs := Inputs{
		Inputs: make(map[string]Input),
		Order:  make([]string, 0),
	}

	inputs.Add(QUIT)
	inputs.Add(SELECT)
	inputs.Add(HELP)

	username := textinput.New()
	username.Placeholder = "Username"
	username.Focus()

	return RegisterPage{
		registerBtn: registerBtn,
		options:     options,
		inputs:      inputs,

		username: username,
		save:     save,
	}
}

func (m RegisterPage) Init() tea.Cmd {
	return nil
}

func (m RegisterPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil
	var model tea.Model = nil

	m.username, cmd = m.username.Update(msg)
	m.registerBtn.username = m.username.Value()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		input, ok := m.inputs.Inputs[msg.String()]
		if ok {
			modelTemp, cmdTemp := input.Action(&m)
			if modelTemp != nil {
				model = modelTemp
			}

			if cmdTemp != nil {
				cmd = tea.Batch(cmd, cmdTemp)
			}
		}
	case error:
		m.errorMessage = msg.Error()

		cmd = tea.Batch(cmd, TimedCmd(5*time.Second, ClearErrorMsg{}))
	case ClearErrorMsg:
		m.errorMessage = ""

	// certificate shenanigans
	case *ca.AllowUser:
		cmd = tea.Batch(cmd, createAndSignCertificate(msg))
	case *ca.CertResponse:
		cmd = tea.Batch(cmd, completeRegistration(m.registerBtn.username, msg))
	case RegistrationSuccess:
		service.UseCertificate(
			settings.CliSettings.CertDir+"yappa.key",
			settings.CliSettings.CertDir+"yappa.crt")
		m.errorMessage = "Good job, you registered!"
		model = NewMainPage(m.save)
	}

	if model == nil {
		model = m
	}

	return model, cmd
}

func (m RegisterPage) View() string {
	s := "\n\n" + m.username.View() + "\n\n"

	s += WhiteForeground.Render(m.options[0].String())

	if m.errorMessage != "" {
		s += Warning.Render("\n\nError: ") + m.errorMessage
	}

	s += "\n\n"

	if m.show {
		for _, v := range m.inputs.Order {
			in := m.inputs.Inputs[v]
			keys := Bold.Render("[" + strings.Join(in.Keys, ", ") + "]")
			s += fmt.Sprintf("%v - %v   ", keys, in.Description)
		}
	}

	s += "\n\n"
	return s
}

func createAndSignCertificate(allowUser *ca.AllowUser) tea.Cmd {
	return func() tea.Msg {
		key, err := service.GeneratePrivKey()
		if err != nil {
			return err
		}

		err = savePemFile(key.Pem, "yappa.key")
		if err != nil {
			return err
		}

		csrPem, err := service.GenerateCSR(key.Key, allowUser.User)
		if err != nil {
			return err
		}

		c, err := service.GetHttp3Client()
		if err != nil {
			return err
		}

		yc := service.RegistrationClient{Client: c}
		certResponse, err := yc.CertificateSignatureRequest(allowUser, csrPem)
		if err != nil {
			return err
		}
		return certResponse
	}
}

func completeRegistration(username string, certResponse *ca.CertResponse) tea.Cmd {
	return func() tea.Msg {
		err := savePemFile(certResponse.Cert, "yappa.crt")
		if err != nil {
			return err
		}

		c, err := service.GetHttp3Client()
		if err != nil {
			return err
		}

		k, err := generateAndSaveKyberKeyPair()
		if err != nil {
			return err
		}

		yc := service.RegistrationClient{Client: c}
		err = yc.CompleteRegistration(username, certResponse, k)
		if err != nil {
			return err
		}

		return RegistrationSuccess{}
	}
}

func generateAndSaveKyberKeyPair() (*mlkem.DecapsulationKey1024, error) {
	key, err := mlkem.GenerateKey1024()
	if err != nil {
		log.Println("Kyber generate error: ", err)
		return nil, errors.New("could not save key")
	}

	key.Bytes()
	keyFile, err := os.OpenFile(settings.CliSettings.CertDir+"dk.key", os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Create file error: ", err)
		return nil, errors.New("could not save key")
	}
	defer keyFile.Close()
	pubFile, err := os.OpenFile(settings.CliSettings.CertDir+"dk.pub", os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Public key create error: ", err)
		return nil, errors.New("could not save key")
	}
	defer pubFile.Close()

	_, err = keyFile.Write(key.Bytes())
	if err != nil {
		log.Println("Pem write error: ", err)
		os.Remove(settings.CliSettings.CertDir + "dk.key")
		return nil, errors.New("could not save key")
	}
	_, err = pubFile.Write(key.EncapsulationKey().Bytes())
	if err != nil {
		log.Println("Pem write error: ", err)
		os.Remove(settings.CliSettings.CertDir + "dk.pub")
		return nil, errors.New("could not save key")
	}

	return key, nil
}

func savePemFile(pemBytes []byte, file string) error {
	pem, err := os.OpenFile(settings.CliSettings.CertDir+file, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Pem create error: ", err)
		return errors.New("could not save certificate")
	}
	defer pem.Close()

	_, err = pem.Write(pemBytes)
	if err != nil {
		log.Println("Pem write error: ", err)
		os.Remove(settings.CliSettings.CertDir + file)
		return errors.New("could not save certificate")
	}

	return nil
}
