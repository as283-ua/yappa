package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type Option interface {
	fmt.Stringer
	Select() (tea.Model, tea.Cmd)
}

type Exit struct{}

func (c Exit) Select() (tea.Model, tea.Cmd) {
	return nil, tea.Quit
}

func (c Exit) String() string {
	return "Exit"
}

type Register struct{}

func (c Register) Select() (tea.Model, tea.Cmd) {
	return nil, nil
}

func (c Register) String() string {
	return "Register!"
}
