package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/rand/v2"
	"os"
	"strings"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/internal/client/settings"
	tea "github.com/charmbracelet/bubbletea"
)

type MainPage struct {
	titleScreen string
	cursor      int

	options []Option
	show    bool
	inputs  Inputs

	save *cli_proto.SaveState
}

func (m MainPage) GetOptions() []Option {
	return m.options
}

func (m MainPage) GetSelected() Option {
	return m.options[m.cursor]
}

func (m *MainPage) Up() {
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.options) - 1
	}
}

func (m *MainPage) Down() {
	m.cursor++
	if m.cursor >= len(m.options) {
		m.cursor = 0
	}
}

func (m MainPage) GetInputs() Inputs {
	return m.inputs
}

func (m MainPage) ToggleShow() Inputer {
	m.show = !m.show
	return m
}

func (m MainPage) Save() *cli_proto.SaveState {
	return m.save
}

func NewMainPage(save *cli_proto.SaveState) MainPage {
	if save == nil {
		log.Println("nil save state")
		save = &cli_proto.SaveState{}
	}

	titleScreen := Titles[rand.Int()%len(Titles)]
	lines := strings.Count(titleScreen, "\n") + 1
	totalLines := 15

	for i := 0; i < (totalLines-lines)/2; i++ {
		titleScreen = "\n" + titleScreen
	}

	for i := lines + (totalLines-lines)/2; i < totalLines; i++ {
		titleScreen += "\n"
	}

	options := make([]Option, 0, 2)

	if !hasCert() {
		options = append(options, GoToRegister{})
	} else {
		options = append(options, GoToUsersPage{})
	}

	options = append(options, Exit{})

	inputs := Inputs{
		Inputs: make(map[string]Input),
		Order:  make([]string, 0),
	}

	inputs.Add(DOWN)
	inputs.Add(UP)
	inputs.Add(QUIT)
	inputs.Add(SELECT)
	inputs.Add(HELP)

	return MainPage{
		options:     options,
		titleScreen: titleScreen,
		inputs:      inputs,
		save:        save,
	}
}

func (m MainPage) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m MainPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil
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

func (m MainPage) View() string {
	s := ""

	s += m.titleScreen + "\n\n"

	for i, option := range m.options {
		if i == m.cursor {
			s += WhiteForeground.Render("> ", option.String())
		} else {
			s += "  " + option.String()
		}
		s += "\n\n"
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

func hasCert() bool {
	_, err := os.Stat(settings.CliSettings.CertDir + "yappa.crt")
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return false

}
