package ui

import (
	"crypto/mlkem"
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
	return fmt.Sprintf("%s - %s (serial %v)\n%s\n", senderStyle.Render(m.Sender), t.Format("2 Jan 2006 15:04:05"), m.Serial, msg.Message.Msg)
}

type ChatPage struct {
	peer         *server.UserData
	encapKey     *mlkem.EncapsulationKey1024
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

	subId        int
	subscription <-chan *client.ClientEvent
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

	vp := viewport.New(120, 20)
	vp.KeyMap.Down.SetKeys("down")
	vp.KeyMap.HalfPageDown.SetKeys("ctrl+down")
	vp.KeyMap.PageDown.SetKeys("pgdown")
	vp.KeyMap.Up.SetKeys("up")
	vp.KeyMap.HalfPageUp.SetKeys("ctrl+up")
	vp.KeyMap.PageUp.SetKeys("pgup")

	return ChatPage{
		save:      save,
		prev:      prev,
		peer:      user,
		viewport:  vp,
		textbox:   textbox,
		selfStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8")),
		peerStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#45f")),
		inputs:    inputs,
		chat:      nil,
		subId:     -1,
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

func loadChat(saveState *client.SaveState, peer *server.UserData) tea.Cmd {
	return func() tea.Msg {
		log.Println("Loading chat")
		chat := save.DirectChatByUser(saveState, peer.Username)
		var err error
		if chat == nil {
			log.Println("First time chatting, retrieving data...")
			chat, err = service.GetChatClient().NewChat(peer)
			if err != nil {
				log.Printf("Error creating chat: %v", err)
				return err
			}
		}
		return chat
	}
}

func (m ChatPage) waitMessage() tea.Msg {
	msg := <-m.subscription
	return msg
}

func (m ChatPage) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, loadChat(m.save, m.peer), m.waitMessage)
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

		var err error
		m.encapKey, err = mlkem.NewEncapsulationKey1024(m.chat.Peer.KeyExchange)
		if err != nil {
			cmd = tea.Batch(cmd, func() tea.Msg { return err })
			break
		}

		m.vpContent = ""
		text := ""
		for _, ev := range m.chat.Events {
			if ev.Sender == service.GetUsername() {
				text = messageToString(ev, m.selfStyle)
			} else {
				text = messageToString(ev, m.peerStyle)
			}
			if text != "" {
				m.vpContent += text + "\n"
			}
		}
		m.viewport.SetContent(m.vpContent)
		m.viewport.GotoBottom()

		subId, subscription := service.GetChatClient().Subscribe([32]byte(m.chat.Peer.InboxId))
		if m.subId < 0 {
			service.GetChatClient().Unsubscribe([32]byte(m.chat.Peer.InboxId), m.subId)
		}
		m.subId = subId
		m.subscription = subscription
		cmd = tea.Batch(cmd, m.waitMessage)
	case MsgSend:
		txt := strings.TrimSpace(m.textbox.Value())
		if txt == "" {
			break
		}
		encMsg, event, err := service.EncryptMessageForPeer(m.chat, txt)
		if err != nil {
			cmd = tea.Batch(cmd, func() tea.Msg { return err })
			break
		}

		err = service.GetChatClient().Send(&server.ClientMessage{
			Payload: &server.ClientMessage_Send{
				Send: encMsg,
			},
		})
		if err != nil {
			cmd = tea.Batch(cmd, func() tea.Msg { return err })
			break
		}
		save.NewEvent(m.chat, m.chat.CurrentSerial+1, service.Ratchet(m.chat.Key), event)

		eventIdx := event.Serial - m.chat.SerialStart
		var keyRotUserOffset uint64 = 0
		if m.peer.Username == m.chat.Initiator {
			keyRotUserOffset = service.MLKEM_RATCHET_INTERVAL / 2
		}
		if (eventIdx+keyRotUserOffset-1)%service.MLKEM_RATCHET_INTERVAL == 0 {
			encMsg, event, key, err := service.KeyExchangeEvent(m.chat, m.encapKey)
			if err != nil {
				cmd = tea.Batch(cmd, func() tea.Msg { return err })
				break
			}
			err = service.GetChatClient().Send(&server.ClientMessage{
				Payload: &server.ClientMessage_Send{
					Send: encMsg,
				},
			})
			if err != nil {
				cmd = tea.Batch(cmd, func() tea.Msg { return err })
				break
			}
			save.NewEvent(m.chat, m.chat.CurrentSerial+1, key, event)
		}
		m.textbox.SetValue("")
		msgTxt := messageToString(event, m.selfStyle)
		if msgTxt != "" {
			m.vpContent += msgTxt + "\n"
			m.viewport.SetContent(m.vpContent)
			m.viewport.GotoBottom()
		}
	case *client.ClientEvent:
		msgTxt := messageToString(msg, m.peerStyle)
		if msgTxt != "" {
			goToBottom := false
			if m.viewport.AtBottom() {
				goToBottom = true
			}
			m.vpContent += msgTxt + "\n"
			m.viewport.SetContent(m.vpContent)
			if goToBottom {
				m.viewport.GotoBottom()
			}
		}
		cmd = tea.Batch(cmd, m.waitMessage)
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
	if m.chat != nil {
		s += fmt.Sprintf("Inbox id: %v\n", m.chat.Peer.InboxId)
		s += fmt.Sprintf("Current expected message serial: %v\n", m.chat.CurrentSerial)
		s += fmt.Sprintf("Initiator: %v\n", m.chat.Initiator)
		s += fmt.Sprintf("%v ... %v\n", m.chat.Key[:5], m.chat.Key[len(m.chat.Key)-5:])
	}

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
