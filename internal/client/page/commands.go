package page

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func SendTimedMessage(msg tea.Msg, t time.Duration) tea.Cmd {
	return func() tea.Msg {
		timer := time.NewTimer(t)
		<-timer.C

		return msg
	}
}

type TickMsg time.Time

func TimedCmd(duration time.Duration, msg tea.Msg) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return msg
	})
}
