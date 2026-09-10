package main

import (
	"fmt"
	"os"
	"tock/db"
	"tock/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer database.Close()

	app, err := ui.NewApp(database)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init app:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
