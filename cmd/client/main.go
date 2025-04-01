package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/as283-ua/yappa/internal/client/page"
	"github.com/as283-ua/yappa/internal/client/settings"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	certDir *string
)

func main() {
	certDir = flag.String("certs", "certs/client", "Path to certs directory")

	flag.Parse()

	settings.CliSettings = settings.Settings{
		CertDir: *certDir,
	}

	p := tea.NewProgram(page.NewMainPage())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
