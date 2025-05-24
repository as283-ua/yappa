package ui

import (
	"fmt"
	"log"
	"math"
	"time"

	cli_proto "github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/service"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type UsernameList []string

type NewUserOpt struct {
	username string
}

func (r NewUserOpt) String() string {
	return r.username
}

func (r NewUserOpt) Select(_ *cli_proto.SaveState) (tea.Model, tea.Cmd) {
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

type FindPage struct {
	search  textinput.Model
	users   []Option
	userSet map[string]bool

	cursor       int
	errorMessage string

	inputs Inputs
	show   bool

	save *cli_proto.SaveState

	prev tea.Model
}

func (m FindPage) GetOptions() []Option {
	return m.users
}

func (m FindPage) GetSelected() Option {
	return m.users[m.cursor]
}

func (m *FindPage) Up() {
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.users) - 1
	}
}

func (m *FindPage) Down() {
	m.cursor++
	if m.cursor >= len(m.users) {
		m.cursor = 0
	}
}

func (m FindPage) GetInputs() Inputs {
	return m.inputs
}

func (m FindPage) ToggleShow() Inputer {
	m.show = !m.show
	return m
}

func (m FindPage) Shows() bool {
	return m.show
}

func (m FindPage) Save() *cli_proto.SaveState {
	return m.save
}

func (m FindPage) Previous() tea.Model {
	return m.prev
}

var RefreshFindChats = Input{
	Keys:        []string{"ctrl+r"},
	Description: "Refresh",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		findPage, ok := m.(*FindPage)
		if !ok {
			return m, nil
		}

		return findPage, findPage.Init()
	},
}

func NewFindPage(save *cli_proto.SaveState, prev tea.Model) FindPage {
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
	inputs.Add(RETURN)
	inputs.Add(RefreshFindChats)
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

	search := textinput.New()
	search.Placeholder = "Search user..."
	search.Prompt = "◆ "
	search.Focus()

	return FindPage{
		search:       search,
		users:        make([]Option, 0, 10),
		userSet:      make(map[string]bool),
		inputs:       inputs,
		show:         false,
		errorMessage: "",
		save:         save,
		prev:         prev,
	}
}

func (m FindPage) Init() tea.Cmd {
	return func() tea.Msg {
		c, err := service.GetHttp3Client()
		if err != nil {
			return err
		}

		yc := service.UsersClient{Client: c}
		usernames, err := yc.GetUsers(0, 10, "")
		if err != nil {
			return err
		}
		var res UsernameList = usernames
		return res
	}
}

func (m FindPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case UsernameList:
		for _, v := range msg {
			if _, ok := m.userSet[v]; !ok {
				m.users = append(m.users, NewUserOpt{username: v})
				m.userSet[v] = true
			}
		}
	case *server.UserData:
		model = NewChatPage(m.save, msg)
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

func (m FindPage) View() string {
	s := `▀██▀▀▀▀█  ██               ▀██                                            
 ██  ▄   ▄▄▄  ▄▄ ▄▄▄     ▄▄ ██     ▄▄▄ ▄▄▄   ▄▄▄▄    ▄▄▄▄  ▄▄▄ ▄▄   ▄▄▄▄  
 ██▀▀█    ██   ██  ██  ▄▀  ▀██      ██  ██  ██▄ ▀  ▄█▄▄▄██  ██▀ ▀▀ ██▄ ▀  
 ██       ██   ██  ██  █▄   ██      ██  ██  ▄ ▀█▄▄ ██       ██     ▄ ▀█▄▄ 
▄██▄     ▄██▄ ▄██▄ ██▄ ▀█▄▄▀██▄     ▀█▄▄▀█▄ █▀▄▄█▀  ▀█▄▄▄▀ ▄██▄    █▀▄▄█▀ `

	s += "\n\n\n" + m.search.View() + "\n\n\n"

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
