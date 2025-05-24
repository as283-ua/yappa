package ui

import (
	"fmt"
	"time"

	"github.com/as283-ua/yappa/api/gen/client"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func MessageToString(m *client.ClientEvent, senderStyle lipgloss.Style) string {
	t := time.Unix(int64(m.Timestamp), 0).UTC()
	fmt.Println(t.Format("2 Jan 2006 15:04:05"))
	return fmt.Sprintf("%s - %s\n%s\n", senderStyle.Render("@"+m.Sender), t.Format("2 Jan 2006 15:04:05"), "Message goes here")
}

type ChatPage struct {
	username     string
	viewport     viewport.Model
	textbox      textarea.Model
	msg          string
	errorMessage string

	senderStyle    lipgloss.Style
	recipientStyle lipgloss.Style

	inputs Inputs
	show   bool

	save *client.SaveState
}

func NewChatPage(save *client.SaveState, user *server.UserData) ChatPage {
	textbox := textarea.New()
	textbox.Focus()
	textbox.Placeholder = "Send a message..."
	textbox.Prompt = "┃ "
	textbox.CharLimit = 280
	textbox.ShowLineNumbers = false
	textbox.SetHeight(5)
	textbox.SetWidth(80)
	textbox.FocusedStyle.CursorLine = lipgloss.NewStyle()

	return ChatPage{
		save:           save,
		username:       user.Username,
		viewport:       viewport.New(80, 12),
		textbox:        textbox,
		senderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8")),
		recipientStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#45f")),
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

func (m ChatPage) Init() tea.Cmd {
	return nil
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

	s = fmt.Sprintf("Chat with '%s'\n", m.username)

	s += "_________________________\n"
	s += m.viewport.View() + "\n"
	s += "‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾\n\n"
	s += m.textbox.View() + "\n"
	s += "ctrl+s to post\n"

	if m.msg != "" {
		s += fmt.Sprintf("Info: %s\n\n", m.msg)
	}

	return s
}
