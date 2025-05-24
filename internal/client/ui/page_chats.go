package ui

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type UserChatOpt struct {
	username string
}

func (r UserChatOpt) String() string {
	return r.username
}

func (r UserChatOpt) Select(_ *cli_proto.SaveState) (tea.Model, tea.Cmd) {
	return nil, func() tea.Msg {
		return errors.New("not implemented")
	}
}

type UsersPage struct {
	search textinput.Model
	users  []Option

	cursor       int
	errorMessage string

	inputs Inputs
	show   bool

	save *cli_proto.SaveState
}

func (m UsersPage) GetOptions() []Option {
	return m.users
}

func (m UsersPage) GetSelected() Option {
	return m.users[m.cursor]
}

func (m *UsersPage) Up() {
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.users) - 1
	}
}

func (m *UsersPage) Down() {
	m.cursor++
	if m.cursor >= len(m.users) {
		m.cursor = 0
	}
}

func (m UsersPage) GetInputs() Inputs {
	return m.inputs
}

func (m UsersPage) ToggleShow() Inputer {
	m.show = !m.show
	return m
}

func (m UsersPage) Save() *cli_proto.SaveState {
	return m.save
}

func NewUsersPage(save *cli_proto.SaveState) UsersPage {
	if save == nil {
		log.Println("nil save state")
		save = &cli_proto.SaveState{}
	}

	inputs := Inputs{
		Inputs: make(map[string]Input),
		Order:  make([]string, 0),
	}

	inputs.Add(DOWN)
	inputs.Add(UP)
	inputs.Add(QUIT)
	inputs.Add(SELECT)
	inputs.Add(HELP)

	textbox := textarea.New()
	textbox.Focus()
	textbox.Placeholder = "Send a message..."
	textbox.Prompt = "┃ "
	textbox.CharLimit = 280
	textbox.ShowLineNumbers = false
	textbox.SetHeight(5)
	textbox.SetWidth(80)

	seach := textinput.New()
	seach.Placeholder = "Seach user..."
	seach.Focus()

	return UsersPage{
		search: seach,
		users: []Option{
			UserChatOpt{username: "andrejs"},
			UserChatOpt{username: "user1"},
			UserChatOpt{username: "test"},
			UserChatOpt{username: "1234"},
			UserChatOpt{username: "some_guy"},
			UserChatOpt{username: "another_one"},
		},
		inputs:       inputs,
		show:         false,
		errorMessage: "",
		save:         save,
	}
}

func (m UsersPage) Init() tea.Cmd {
	return nil
}

func (m UsersPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil
	var model tea.Model = nil

	m.search, cmd = m.search.Update(msg)

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
	}

	if model == nil {
		model = m
	}

	return model, cmd
}

const MAX_VISIBLE_USERS = 5

func (m UsersPage) View() string {
	s := "\n\n" + m.search.View() + "\n\n\n"

	boundUp := m.cursor - 2
	boundDown := m.cursor + 2
	if boundUp < 0 {
		boundUp = 0
		boundDown = MAX_VISIBLE_USERS - 1
	}

	if boundDown >= len(m.users) {
		boundDown = len(m.users) - 1
		boundUp = len(m.users) - 5
	}

	if boundUp != 0 {
		s += "▲ ▲ ▲\n"
	}

	for idx, v := range m.users {
		if idx < boundUp || idx > boundDown {
			continue
		}
		if m.cursor == idx {
			s += WhiteForeground.Render(v.String()) + "\n\n"
		} else {
			s += v.String() + "\n\n"
		}
	}
	if boundDown != len(m.users)-1 {
		s += "▼ ▼ ▼\n"
	}

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
