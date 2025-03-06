package page

import (
	"math/rand/v2"

	"github.com/as283-ua/yappa/internal/client/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MainPage struct {
	titleScreen string
	cursor      int
	options     []string
}

func NewMainPage() MainPage {
	return MainPage{
		options: []string{
			"Exit",
		},
		titleScreen: ui.Titles[rand.Int()%len(ui.Titles)],
	}
}

func (m MainPage) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m MainPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, tea.Quit
		case "enter", "right":
			return m, nil
		}
	}
	return m, nil
}

var WhiteForeground = lipgloss.NewStyle().Foreground(lipgloss.Color("#000")).Background(lipgloss.Color("#FFF"))

func (m MainPage) View() string {
	s := ""

	s += m.titleScreen + "\n\n"

	for i, option := range m.options {
		if i == m.cursor {
			s += WhiteForeground.Render("> ", option)
		} else {
			s += "  " + option
		}
		s += "\n"
	}
	return s
}
