package ui

import tea "github.com/charmbracelet/bubbletea"

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
	Keys:        []string{"q", "ctrl+c"},
	Description: "Quit",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		return nil, tea.Quit
	},
}

var SELECT = Input{
	Keys:        []string{"enter", "right"},
	Description: "Select",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		optioner, ok := m.(Optioner)
		if !ok {
			return m, nil
		}
		return optioner.GetSelected().Select()
	},
}

var HELP = Input{
	Keys:        []string{"h"},
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
