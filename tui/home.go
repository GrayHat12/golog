package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	homeTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#d4bbff"))
	homeLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb1c3")).Width(12)
	homeValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7fd0ff")).Bold(true)
	homeHelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e1f1"))
	homeSuccStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	homeErrStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb4ab"))
	homeBreakStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd9e0")).Italic(true)
	homeMarginStyle = lipgloss.NewStyle().Margin(2, 4)
	homePathStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#626273")).Italic(true)
)

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) updateHome(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Keep ticking as long as we stay on the home screen.
		return m, doTick()

	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			m.screen = screenAddEntry
			m.statusMsg = ""
			m.err = nil
			m.addEntry = newAddEntryModel()
			var cmd tea.Cmd
			m.addEntry, cmd = m.addEntry.focusName()
			return m, cmd

		case "b":
			if m.cfg.OnAddBreak != nil {
				if err := m.cfg.OnAddBreak(); err != nil {
					m.err = err
					m.statusMsg = ""
				} else {
					m.statusMsg = "Break started"
					m.err = nil
				}
			}
			return m, nil

		case "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) viewHome() tea.View {
	name := ""
	if m.cfg.CurrentName != nil {
		name = m.cfg.CurrentName()
	}

	var elapsed time.Duration
	if m.cfg.CurrentStart != nil {
		if start := m.cfg.CurrentStart(); !start.IsZero() {
			elapsed = time.Since(start)
		}
	}

	// ── Header ────────────────────────────────────────────────────────────────
	out := homeTitleStyle.Render("⏱  Time Tracker") + "\n\n"

	// ── Currently ─────────────────────────────────────────────────────────────
	out += homeLabelStyle.Render("Currently")
	if name == "" {
		out += homeBreakStyle.Render("— on break —")
	} else {
		out += homeValueStyle.Render(name)
	}
	out += "\n"

	// ── Elapsed ───────────────────────────────────────────────────────────────
	out += homeLabelStyle.Render("Elapsed")
	if elapsed == 0 && name == "" {
		out += homeBreakStyle.Render("—")
	} else {
		// fmt.Printf("elapsed %f", elapsed.Seconds())
		out += homeValueStyle.Render(formatDuration(elapsed))
	}
	out += "\n"

	// ── Status / error ────────────────────────────────────────────────────────
	if m.statusMsg != "" {
		out += "\n" + homeSuccStyle.Render("✓  "+m.statusMsg)
	}
	if m.err != nil {
		out += "\n" + homeErrStyle.Render("✗  "+m.err.Error())
	}

	// ── Journal Path ──────────────────────────────────────────────────────────
	if m.cfg.JournalPath != "" {
		out += "\n\n" + homePathStyle.Render("📁 "+m.cfg.JournalPath)
	} else {
		out += "\n\n"
	}

	// ── Help bar ──────────────────────────────────────────────────────────────
	out += "\n\n" + homeHelpStyle.Render(
		"n  new entry    b  break    esc / ctrl+c  quit",
	)

	return tea.NewView(homeMarginStyle.Render(out))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	min := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, min, sec)
	}
	return fmt.Sprintf("%dm %02ds", min, sec)
}
