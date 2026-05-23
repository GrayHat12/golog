package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// screenID identifies which screen is currently rendered.
type screenID int

const (
	screenHome screenID = iota
	screenAddEntry
)

// tickMsg is fired every second to refresh the elapsed-time counter.
type tickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// model is the root BubbleTea model that owns all screen state.
type model struct {
	cfg    Config
	screen screenID
	width  int
	height int

	// Transient status shown on the home screen after an action.
	statusMsg string
	err       error

	// Screen sub-models.
	addEntry addEntryModel
}

func newModel(cfg Config) model {
	return model{
		cfg:      cfg,
		screen:   screenHome,
		addEntry: newAddEntryModel(),
	}
}

// ── tea.Model interface ───────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	// Start the per-second ticker immediately.
	return doTick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global hard-quit.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.screen {
	case screenHome:
		return m.updateHome(msg)
	case screenAddEntry:
		return m.updateAddEntry(msg)
	}
	return m, nil
}

func (m model) View() tea.View {
	switch m.screen {
	case screenHome:
		return m.viewHome()
	case screenAddEntry:
		return m.viewAddEntry()
	}
	return tea.NewView("")
}
