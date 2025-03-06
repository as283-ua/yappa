package main

import (
	"fmt"
	"os"

	"github.com/as283-ua/yappa/internal/client/page"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(page.NewMainPage())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
