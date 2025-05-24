package ui

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
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

var FindChats = Input{
	Keys:        []string{"ctrl+n"},
	Description: "Find new chats",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		userpage, ok := m.(*UsersPage)
		if !ok {
			log.Fatalf("%t", m)
			return m, nil
		}
		newPage := NewFindPage(userpage.save)

		return newPage, newPage.Init()
	},
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
	inputs.Add(FindChats)
	inputs.Add(QUIT)
	inputs.Add(SELECT)
	inputs.Add(HELP)

	search := textinput.New()
	search.Placeholder = "Seach user..."
	search.Prompt = "◆ "
	search.Focus()

	users := make([]Option, len(save.Chats))
	for _, chat := range save.Chats {
		users = append(users, UserChatOpt{username: chat.Peer.Username})
	}

	return UsersPage{
		search:       search,
		users:        users,
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
	s := `    █               ▄    ██                              ▀██                ▄          
   ███      ▄▄▄▄  ▄██▄  ▄▄▄  ▄▄▄▄ ▄▄▄   ▄▄▄▄       ▄▄▄▄   ██ ▄▄    ▄▄▄▄   ▄██▄   ▄▄▄▄  
  █  ██   ▄█   ▀▀  ██    ██   ▀█▄  █  ▄█▄▄▄██    ▄█   ▀▀  ██▀ ██  ▀▀ ▄██   ██   ██▄ ▀  
 ▄▀▀▀▀█▄  ██       ██    ██    ▀█▄█   ██         ██       ██  ██  ▄█▀ ██   ██   ▄ ▀█▄▄ 
▄█▄  ▄██▄  ▀█▄▄▄▀  ▀█▄▀ ▄██▄    ▀█     ▀█▄▄▄▀     ▀█▄▄▄▀ ▄██▄ ██▄ ▀█▄▄▀█▀  ▀█▄▀ █▀▄▄█▀ `

	s += "\n\n" + m.search.View() + "\n\n"

	boundUp := m.cursor - 2
	boundDown := m.cursor + 2
	if boundUp < 0 {
		boundUp = 0
		boundDown = int(math.Min(float64(len(m.users)-1), MAX_VISIBLE_USERS-1))
	}

	if boundDown >= len(m.users) {
		boundDown = int(math.Max(float64(len(m.users)-1), 0))
		boundUp = int(math.Max(float64(len(m.users)-5), 0))
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

	if len(m.users) == 0 {
		s += WhiteForeground.Render("No active chats") + "\n"
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
