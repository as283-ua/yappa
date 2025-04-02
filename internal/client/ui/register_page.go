package ui

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/client/service"
	"github.com/as283-ua/yappa/internal/client/settings"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Register struct {
	username string
}

func (r Register) String() string {
	return "Register"
}

func (r Register) Select() (tea.Model, tea.Cmd) {
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

type RegisterPage struct {
	username textinput.Model

	registerBtn *Register
	options     []Option
	show        bool
	inputs      Inputs

	errorMessage string
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

func NewRegisterPage() RegisterPage {
	options := make([]Option, 0, 2)

	registerBtn := &Register{username: ""}

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
	case *gen.AllowUser:
		cmd = tea.Batch(cmd, proceedWithRegistration(msg))
	case *gen.CertResponse:
		m.errorMessage = "Good job, you registered!"
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

func proceedWithRegistration(allowUser *gen.AllowUser) tea.Cmd {
	return func() tea.Msg {
		key, err := service.GeneratePrivKey()
		if err != nil {
			return err
		}

		err = saveKeyFile(allowUser.User, key.Pem)
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

func saveKeyFile(username string, privPem []byte) error {
	pem, err := os.OpenFile(settings.CliSettings.CertDir+username+".key", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Println("Pem create error: ", err)
		return errors.New("could not save private key")
	}
	defer pem.Close()
	_, err = pem.Write(privPem)
	if err != nil {
		log.Println("Pem write error: ", err)
		os.Remove(settings.CliSettings.CertDir + username + ".key")
		return errors.New("could not save private key")
	}
	pem.Chmod(0400)

	return nil
}
