package ui

import (
	"fmt"
	"strings"

	"github.com/as283-ua/yappa/api/gen/client"
	tea "github.com/charmbracelet/bubbletea"
)

type Stateful interface {
	Save() *client.SaveState
}

type Optioner interface {
	tea.Model
	GetOptions() []Option
	GetSelected() Option
	Up()
	Down()
}

type Inputer interface {
	tea.Model
	GetInputs() Inputs
	ToggleShow() Inputer
	Shows() bool
}

type Returner interface {
	Previous() tea.Model
}

type Input struct {
	Keys        []string
	Description string
	Action      func(tea.Model) (tea.Model, tea.Cmd)
}

type Inputs struct {
	Inputs map[string]Input
	Order  []string
}

func (i *Inputs) Add(in Input) {
	if len(in.Keys) == 0 {
		return
	}

	i.Order = append(i.Order, in.Keys[0])
	for _, v := range in.Keys {
		i.Inputs[v] = in
	}
}

var DOWN = Input{
	Keys:        []string{"down"},
	Description: "Move cursor down",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		optioner, ok := m.(Optioner)
		if !ok {
			return m, nil
		}
		optioner.Down()
		return optioner, nil
	},
}

var UP = Input{
	Keys:        []string{"up"},
	Description: "Move cursor up",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		optioner, ok := m.(Optioner)
		if !ok {
			return m, nil
		}
		optioner.Up()
		return optioner, nil
	},
}

var QUIT = Input{
	Keys:        []string{"ctrl+c"},
	Description: "Quit",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		return nil, tea.Quit
	},
}

var SELECT = Input{
	Keys:        []string{"enter"},
	Description: "Select",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		state, ok := m.(Stateful)
		if !ok {
			return m, nil
		}
		save := state.Save()
		optioner, ok := m.(Optioner)
		if !ok {
			return m, nil
		}
		if len(optioner.GetOptions()) == 0 {
			return m, nil
		}
		return optioner.GetSelected().Select(save)
	},
}

var HELP = Input{
	Keys:        []string{"ctrl+h"},
	Description: "Display controls",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		inputer, ok := m.(Inputer)
		if !ok {
			return m, nil
		}
		inputer = inputer.ToggleShow()
		return inputer, nil
	},
}

var RETURN = Input{
	Keys:        []string{"ctrl+z", "ctrl+left"},
	Description: "Return to previous page",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		returner, ok := m.(Returner)
		if !ok {
			return m, nil
		}
		if returner.Previous() == nil {
			return m, nil
		}
		return returner.Previous(), returner.Previous().Init()
	},
}

func Render(i Inputer) string {
	s := ""
	chars := 0
	if i.Shows() {
		for _, v := range i.GetInputs().Order {
			in := i.GetInputs().Inputs[v]
			keys := Bold.Render("[" + strings.Join(in.Keys, ", ") + "]")
			entry := fmt.Sprintf("%v - %v   ", keys, in.Description)
			s += entry

			chars += len(entry)
			if chars >= 120 {
				s += "\n"
				chars = 0
			}
		}
	}
	return s
}
