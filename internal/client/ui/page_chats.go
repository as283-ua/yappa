package ui

import (
	"fmt"
	"log"
	"math"
	"time"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/service"
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
		c, err := service.GetHttp3Client()
		if err != nil {
			return err
		}

		yc := service.UsersClient{Client: c}
		user, err := yc.GetUserData(r.username)
		if err != nil {
			return err
		}

		return user
	}
}

var FindChats = Input{
	Keys:        []string{"ctrl+n", "ctrl+d"},
	Description: "Find new chats",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		userpage, ok := m.(*ActiveChatsPage)
		if !ok {
			return m, nil
		}
		newPage := NewFindPage(userpage.save, m)

		return newPage, newPage.Init()
	},
}

type ActiveChatsPage struct {
	search textinput.Model
	users  []Option

	cursor       int
	errorMessage string

	inputs Inputs
	show   bool

	save *cli_proto.SaveState
	prev tea.Model
}

func (m ActiveChatsPage) GetOptions() []Option {
	return m.users
}

func (m ActiveChatsPage) GetSelected() Option {
	return m.users[m.cursor]
}

func (m *ActiveChatsPage) Up() {
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.users) - 1
	}
}

func (m *ActiveChatsPage) Down() {
	m.cursor++
	if m.cursor >= len(m.users) {
		m.cursor = 0
	}
}

func (m ActiveChatsPage) GetInputs() Inputs {
	return m.inputs
}

func (m ActiveChatsPage) ToggleShow() Inputer {
	m.show = !m.show
	return m
}

func (m ActiveChatsPage) Shows() bool {
	return m.show
}

func (m ActiveChatsPage) Save() *cli_proto.SaveState {
	return m.save
}

func (m ActiveChatsPage) Previous() tea.Model {
	return m.prev
}

func NewActiveChatsPage(save *cli_proto.SaveState, prev tea.Model) ActiveChatsPage {
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
	inputs.Add(RETURN)
	inputs.Add(QUIT)
	inputs.Add(SELECT)
	inputs.Add(HELP)

	search := textinput.New()
	search.Placeholder = "Search user..."
	search.Prompt = "◆ "
	search.Focus()

	users := make([]Option, 0, len(save.Chats))
	for _, chat := range save.Chats {
		users = append(users, UserChatOpt{username: chat.Peer.Username})
	}

	log.Println(users)

	return ActiveChatsPage{
		search:       search,
		users:        users,
		inputs:       inputs,
		show:         false,
		errorMessage: "",
		save:         save,
		prev:         prev,
	}
}

func (m ActiveChatsPage) Init() tea.Cmd {
	return nil
}

func (m ActiveChatsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case *server.UserData:
		model = NewChatPage(m.save, m, msg)
		cmd = model.Init()
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

func (m ActiveChatsPage) View() string {
	s := `    █               ▄    ██                              ▀██                ▄          
   ███      ▄▄▄▄  ▄██▄  ▄▄▄  ▄▄▄▄ ▄▄▄   ▄▄▄▄       ▄▄▄▄   ██ ▄▄    ▄▄▄▄   ▄██▄   ▄▄▄▄  
  █  ██   ▄█   ▀▀  ██    ██   ▀█▄  █  ▄█▄▄▄██    ▄█   ▀▀  ██▀ ██  ▀▀ ▄██   ██   ██▄ ▀  
 ▄▀▀▀▀█▄  ██       ██    ██    ▀█▄█   ██         ██       ██  ██  ▄█▀ ██   ██   ▄ ▀█▄▄ 
▄█▄  ▄██▄  ▀█▄▄▄▀  ▀█▄▀ ▄██▄    ▀█     ▀█▄▄▄▀     ▀█▄▄▄▀ ▄██▄ ██▄ ▀█▄▄▀█▀  ▀█▄▀ █▀▄▄█▀ `

	s += "\n\n\n"
	// s += m.search.View() + "\n\n\n"

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
		entry := fmt.Sprintf("%v. %v", idx+1, v.String())
		if m.cursor == idx {
			s += WhiteForeground.Render(entry) + "\n\n"
		} else {
			s += entry + "\n\n"
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

	s += Render(m)

	s += "\n\n"
	return s
}
