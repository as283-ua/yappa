package ui

import (
	"fmt"
	"strings"

	"github.com/as283-ua/yappa/internal/client/service"
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
			return fmt.Errorf("username must be at least 4 characters long")
		}

		c, _ := service.GetHttp3Client()
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

	options []Option
	show    bool
	inputs  Inputs
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

	options = append(options, Register{})

	options = append(options, Exit{})

	inputs := Inputs{
		Inputs: make(map[string]Input),
		Order:  make([]string, 0),
	}

	inputs.Add(QUITTYPABLE)
	inputs.Add(SELECT)
	inputs.Add(HELP)

	username := textinput.New()
	username.Placeholder = "Username"
	username.Focus()

	return RegisterPage{
		options: options,
		inputs:  inputs,

		username: username,
	}
}

func (m RegisterPage) Init() tea.Cmd {
	return nil
}

func (m RegisterPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil
	m.username, cmd = m.username.Update(msg)
	var model tea.Model = m

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
	}

	return model, cmd
}

func (m RegisterPage) View() string {
	s := ""

	s += m.username.View() + "\n"

	s += WhiteForeground.Render("> ", m.options[0].String())

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
