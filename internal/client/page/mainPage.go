package page

import (
	"math/rand/v2"
	"strings"

	"github.com/as283-ua/yappa/internal/client/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type MainPage struct {
	titleScreen string
	cursor      int
	options     []Option
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

	return MainPage{
		options: []Option{
			Exit{},
		},
		titleScreen: titleScreen,
	}
}

func (m MainPage) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m MainPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var model tea.Model = m
	var cmd tea.Cmd = nil

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down":
			m.cursor++
			if m.cursor >= len(m.options) {
				m.cursor = 0
			}
		case "up":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.options) - 1
			}
		case "q", "ctrl+c":
			cmd = tea.Quit
		case "enter", "right":
			var cmdTemp tea.Cmd
			var modelTemp tea.Model
			modelTemp, cmdTemp = m.options[m.cursor].Select()
			if cmdTemp != nil {
				cmd = tea.Batch(cmd, cmdTemp)
			}

			if modelTemp != nil {
				model = modelTemp
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
	return s
}
