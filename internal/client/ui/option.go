package ui

import (
	"fmt"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	tea "github.com/charmbracelet/bubbletea"
)

type Option interface {
	fmt.Stringer
	Select(save *cli_proto.SaveState) (tea.Model, tea.Cmd)
}

type Exit struct{}

func (c Exit) Select(_ *cli_proto.SaveState) (tea.Model, tea.Cmd) {
	return nil, tea.Quit
}

func (c Exit) String() string {
	return "Exit"
}

type GoToRegister struct{}

func (c GoToRegister) Select(save *cli_proto.SaveState) (tea.Model, tea.Cmd) {
	return NewRegisterPage(save), nil
}

func (c GoToRegister) String() string {
	return "Register!"
}

type GoToUsersPage struct{}

func (c GoToUsersPage) Select(save *cli_proto.SaveState) (tea.Model, tea.Cmd) {
	return NewUsersPage(save), nil
}

func (c GoToUsersPage) String() string {
	return "My chats"
}
