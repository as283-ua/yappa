package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/client/save"
	"github.com/as283-ua/yappa/internal/client/service"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func messageToString(m *client.ClientEvent, senderStyle lipgloss.Style) string {
	msg, ok := m.Payload.(*client.ClientEvent_Message)
	if !ok {
		return ""
	}
	t := time.Unix(int64(m.Timestamp), 0).UTC()
	return fmt.Sprintf("%s - %s\n%s\n", senderStyle.Render("@"+m.Sender), t.Format("2 Jan 2006 15:04:05"), msg.Message.Msg)
}

type ChatPage struct {
	peer         *server.UserData
	chat         *client.Chat
	viewport     viewport.Model
	vpContent    string
	textbox      textarea.Model
	errorMessage string

	selfStyle lipgloss.Style
	peerStyle lipgloss.Style

	inputs Inputs
	show   bool

	save *client.SaveState
	prev tea.Model
}

type MsgSend struct{}

var Send = Input{
	Keys:        []string{"enter"},
	Description: "Send message",
	Action: func(m tea.Model) (tea.Model, tea.Cmd) {
		_, ok := m.(*ChatPage)
		if !ok {
			return m, nil
		}
		return m, func() tea.Msg { return MsgSend{} }
	},
}

func NewChatPage(save *client.SaveState, prev tea.Model, user *server.UserData) ChatPage {
	textbox := textarea.New()
	textbox.Focus()
	textbox.Placeholder = "Send a message..."
	textbox.Prompt = "┃ "
	textbox.CharLimit = 1000
	textbox.ShowLineNumbers = false
	textbox.SetHeight(5)
	textbox.SetWidth(80)
	textbox.FocusedStyle.CursorLine = lipgloss.NewStyle()

	inputs := Inputs{
		Inputs: make(map[string]Input),
		Order:  make([]string, 0),
	}
	inputs.Add(Send)
	inputs.Add(RETURN)
	inputs.Add(QUIT)
	inputs.Add(HELP)

	return ChatPage{
		save:      save,
		prev:      prev,
		peer:      user,
		viewport:  viewport.New(120, 20),
		textbox:   textbox,
		selfStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8")),
		peerStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#45f")),
		inputs:    inputs,
		chat:      &client.Chat{},
	}
}

func (m ChatPage) GetInputs() Inputs {
	return m.inputs
}

func (m ChatPage) ToggleShow() Inputer {
	m.show = !m.show
	return m
}

func (m ChatPage) Shows() bool {
	return m.show
}

func (m ChatPage) Save() *client.SaveState {
	return m.save
}

func (m ChatPage) Previous() tea.Model {
	return m.prev
}

func (m ChatPage) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, func() tea.Msg {
		log.Println("Initiating chat")
		chat := save.DirectChat(m.save, m.peer.Username)
		var err error
		if chat == nil {
			log.Println("First time chatting, retrieving data...")
			chat, err = service.GetChatClient().NewChat(m.peer)
			if err != nil {
				log.Printf("Error creating chat: %v", err)
				return err
			}
		}
		return chat
	})
}

func (m ChatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd, cmdVp, cmdTb tea.Cmd = nil, nil, nil
	var model tea.Model = nil

	m.viewport, cmdVp = m.viewport.Update(msg)
	m.textbox, cmdTb = m.textbox.Update(msg)
	cmd = tea.Batch(cmdVp, cmdTb)

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

	case *client.Chat:
		save.NewDirectChat(m.save, msg)
		log.Printf("Loaded chat with %v\n", msg.Peer.Username)
		m.chat = msg

		m.vpContent = ""
		text := ""
		for _, ev := range m.chat.Events {
			if ev.Sender == service.GetUsername() {
				text = messageToString(ev, m.selfStyle)
			} else {
				text = messageToString(ev, m.peerStyle)
			}
			m.vpContent += text + "\n"
		}
		m.viewport.SetContent(m.vpContent)
		m.viewport.GotoBottom()
	case MsgSend:
		txt := strings.TrimSpace(m.textbox.Value())
		if txt == "" {
			break
		}
		event := &client.ClientEvent{
			Timestamp: uint64(time.Now().UTC().Unix()),
			Serial:    m.chat.CurrentSerial,
			Sender:    service.GetUsername(),
			Payload: &client.ClientEvent_Message{
				Message: &client.ChatMessage{
					Msg: txt,
				},
			},
		}
		// service.GetChatClient().Send()
		save.NewEvent(m.save, m.chat, event)
		m.textbox.SetValue("")
		msgTxt := messageToString(event, m.selfStyle)
		if msgTxt != "" {
			m.vpContent += msgTxt + "\n"
			m.viewport.SetContent(m.vpContent)
			m.viewport.GotoBottom()
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

func (m ChatPage) View() string {
	var s string

	s = fmt.Sprintf("Chat with '%s'\n", m.peer.Username)

	s += "________________________________________________________________________________\n"
	s += m.viewport.View() + "\n"
	s += "‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾\n\n"
	s += m.textbox.View() + "\n"

	if m.errorMessage != "" {
		s += Warning.Render("\n\nError: ") + m.errorMessage
	}

	s += "\n\n"

	s += Render(m)

	s += "\n\n"

	return s
}
