package ui

import (
	"sektron/midi"
	"sektron/sequencer"

	tea "github.com/charmbracelet/bubbletea"
)

type UI struct {
	seq sequencer.Sequencer
}

func New(midi *midi.Server) UI {
	return UI{
		seq: sequencer.New(midi),
	}
}

func (u UI) Init() tea.Cmd {
	return nil
}

func (u UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case sequencer.ClockTickMsg:
		u.seq.Pulse()
		return u, u.seq.Clock()

	case tea.KeyMsg:
		switch msg.String() {

		case " ":
			u.seq.TogglePlay()
			return u, u.seq.Clock()

		// These keys should exit the program.
		case "ctrl+c", "q":
			return u, tea.Quit
		}
	}
	return u, nil
}

func (u UI) View() string {
	return u.seq.View()
}
