package page

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/as283-ua/yappa/internal/client/settings"
	"github.com/as283-ua/yappa/internal/client/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type MainPage struct {
	titleScreen string
	cursor      int

	options []ui.Option
	show    bool
	inputs  ui.Inputs
}

func (m MainPage) GetOptions() []ui.Option {
	return m.options
}

func (m MainPage) GetSelected() ui.Option {
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

func (m MainPage) GetInputs() ui.Inputs {
	return m.inputs
}

func (m MainPage) ToggleShow() ui.Inputer {
	m.show = !m.show
	return m
}

func NewMainPage() MainPage {
	titleScreen := ui.Titles[rand.Int()%len(ui.Titles)]
	lines := strings.Count(titleScreen, "\n") + 1
	totalLines := 15

	for i := 0; i < (totalLines-lines)/2; i++ {
		titleScreen = "\n" + titleScreen
	}

	for i := lines + (totalLines-lines)/2; i < totalLines; i++ {
		titleScreen += "\n"
	}

	options := make([]ui.Option, 0, 2)

	if !hasCert() {
		options = append(options, ui.Register{})
	}

	options = append(options, ui.Exit{})

	inputs := ui.Inputs{
		Inputs: make(map[string]ui.Input),
		Order:  make([]string, 0),
	}

	inputs.Add(ui.DOWN)
	inputs.Add(ui.UP)
	inputs.Add(ui.QUIT)
	inputs.Add(ui.SELECT)
	inputs.Add(ui.HELP)

	return MainPage{
		options:     options,
		titleScreen: titleScreen,
		inputs:      inputs,
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
			s += ui.WhiteForeground.Render("> ", option.String())
		} else {
			s += "  " + option.String()
		}
		s += "\n"
	}

	s += "\n\n"

	if m.show {
		for _, v := range m.inputs.Order {
			in := m.inputs.Inputs[v]
			keys := ui.Bold.Render("[" + strings.Join(in.Keys, ", ") + "]")
			s += fmt.Sprintf("%v - %v   ", keys, in.Description)
		}
	}

	s += "\n\n"
	return s
}

func hasCert() bool {
	_, err := os.Stat(settings.CliSettings.CertDir + "/cert.crt")
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return false

}
