package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── Sub-model ─────────────────────────────────────────────────────────────────

type addEntryField int

const (
	fieldName addEntryField = iota
	fieldLabels
)

type addEntryModel struct {
	nameInput   textinput.Model
	labelsInput textinput.Model
	focused     addEntryField

	// Ghost-text suggestions (updated on every keystroke).
	nameSuggestion  string // full suggested name, e.g. "writing unit tests"
	labelSuggestion string // suggested completion for the last tag, e.g. "bubbletea"
}

func newAddEntryModel() addEntryModel {
	nameIn := textinput.New()
	nameIn.Placeholder = "What are you doing ?"
	nameIn.CharLimit = 120

	labelsIn := textinput.New()
	labelsIn.Placeholder = "meet, research, code"
	labelsIn.CharLimit = 240

	return addEntryModel{
		nameInput:   nameIn,
		labelsInput: labelsIn,
		focused:     fieldName,
	}
}

// focusName focuses the name field and returns the blink Cmd.
func (ae addEntryModel) focusName() (addEntryModel, tea.Cmd) {
	ae.nameInput.Blur()
	ae.labelsInput.Blur()
	cmd := ae.nameInput.Focus()
	ae.focused = fieldName
	return ae, cmd
}

// focusLabels moves focus to the labels field.
func (ae addEntryModel) focusLabels(cfg Config) (addEntryModel, tea.Cmd) {
	ae.nameInput.Blur()
	ae.labelsInput.Blur()
	cmd := ae.labelsInput.Focus()
	ae.focused = fieldLabels
	name := strings.TrimSpace(ae.nameInput.Value())
	suggestions := cfg.PrefilLabels(&name)
	labelValue := strings.Join(suggestions, ", ")
	ae.labelsInput.SetValue(labelValue)
	return ae, cmd
}

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	aeTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#d4bbff"))
	aeGhostStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#383844"))
	aeHelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e1f1"))
	aeHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4a4550")).Italic(true)
	aeMarginStyle = lipgloss.NewStyle().Margin(2, 4)
	aeFieldBox    = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#60b3e2")).
			Padding(0, 1)
)

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) updateAddEntry(msg tea.Msg) (tea.Model, tea.Cmd) {
	ae := m.addEntry

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "esc":
			m.screen = screenHome
			m.addEntry = newAddEntryModel()
			return m, doTick()

		case "tab":
			// Accept the ghost-text suggestion for the focused field.
			if ae.focused == fieldName {
				ae = applyNameCompletion(ae, m.cfg)
			} else {
				ae = applyTagCompletion(ae, m.cfg)
			}
			m.addEntry = ae
			return m, nil

		case "enter":
			if ae.focused == fieldName {
				// Move focus to the labels field.
				var cmd tea.Cmd
				ae, cmd = ae.focusLabels(m.cfg)
				m.addEntry = ae
				return m, cmd
			}
			// Submit the form.
			return m.submitEntry()
		}
	}

	// Delegate the message to whichever input is focused, then refresh suggestions.
	var cmd tea.Cmd
	if ae.focused == fieldName {
		ae.nameInput, cmd = ae.nameInput.Update(msg)
		ae.nameSuggestion = nameGhostSuffix(ae.nameInput.Value(), m.cfg)
	} else {
		ae.labelsInput, cmd = ae.labelsInput.Update(msg)
		ae.labelSuggestion = tagGhostSuffix(ae.labelsInput.Value(), m.cfg)
	}
	m.addEntry = ae
	return m, cmd
}

func (m model) submitEntry() (tea.Model, tea.Cmd) {
	ae := m.addEntry
	e := EntryInput{
		Name:   strings.TrimSpace(ae.nameInput.Value()),
		Labels: strings.TrimSpace(ae.labelsInput.Value()),
	}

	if m.cfg.OnAddEntry != nil {
		if err := m.cfg.OnAddEntry(e); err != nil {
			m.err = err
			m.statusMsg = ""
			m.screen = screenHome
			m.addEntry = newAddEntryModel()
			return m, doTick()
		}
	}

	label := e.Name
	if label == "" {
		label = "(unnamed)"
	}
	m.statusMsg = "Added: " + label
	m.err = nil
	m.screen = screenHome
	m.addEntry = newAddEntryModel()
	return m, doTick()
}

// ── Autocomplete helpers ──────────────────────────────────────────────────────

// nameGhostSuffix returns the suffix that should be shown as ghost text for
// the name field (the part of the suggestion that hasn't been typed yet).
func nameGhostSuffix(input string, cfg Config) string {
	if cfg.GetNameSuggestion == nil || cfg.NameSuggestions == nil || input == "" {
		return ""
	}
	sugg := cfg.GetNameSuggestion(input, cfg.NameSuggestions())
	if sugg == "" {
		return ""
	}
	lower := strings.ToLower
	if strings.HasPrefix(lower(sugg), lower(input)) {
		return sugg[len(input):]
	}
	return ""
}

// tagGhostSuffix returns the suffix to show for the last tag being typed.
func tagGhostSuffix(input string, cfg Config) string {
	if cfg.GetTagSuggestion == nil || cfg.LabelSuggestions == nil || input == "" {
		return ""
	}
	sugg := cfg.GetTagSuggestion(input, cfg.LabelSuggestions())
	if sugg == "" {
		return ""
	}
	// sugg is the completion of the last tag; work out how much the user already typed.
	parts := strings.Split(input, ",")
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	if lastPart == "" {
		return ""
	}
	lower := strings.ToLower
	if strings.HasPrefix(lower(sugg), lower(lastPart)) {
		return sugg[len(lastPart):]
	}
	return ""
}

// applyNameCompletion replaces the name input value with the full suggestion.
func applyNameCompletion(ae addEntryModel, cfg Config) addEntryModel {
	if cfg.AutocompleteName == nil || cfg.NameSuggestions == nil {
		return ae
	}
	completed := cfg.AutocompleteName(ae.nameInput.Value(), cfg.NameSuggestions())
	if completed != "" {
		ae.nameInput.SetValue(completed)
		// Move cursor to end.
		ae.nameInput.CursorEnd()
		ae.nameSuggestion = ""
	}
	return ae
}

// applyTagCompletion replaces the labels input value with the autocompleted version.
func applyTagCompletion(ae addEntryModel, cfg Config) addEntryModel {
	if cfg.AutocompleteLastTag == nil || cfg.LabelSuggestions == nil {
		return ae
	}
	completed := cfg.AutocompleteLastTag(ae.labelsInput.Value(), cfg.LabelSuggestions())
	if completed != "" {
		ae.labelsInput.SetValue(completed)
		ae.labelsInput.CursorEnd()
		ae.labelSuggestion = ""
	}
	return ae
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) viewAddEntry() tea.View {
	ae := m.addEntry

	ae.nameInput.SetWidth(m.width / 2)
	ae.labelsInput.SetWidth(m.width / 2)

	out := aeTitleStyle.Render("＋ New Entry")

	out = lipgloss.JoinVertical(lipgloss.Left, out)

	// ── Name field ────────────────────────────────────────────────────────────
	// nameLabel := aeLabelStyle.Render("Name")
	nameBox := aeFieldBox.Width(m.width/2).Align(lipgloss.Left, lipgloss.Center).Border(lipgloss.Border{
		Top:          fmt.Sprintf(" Activity %s", strings.Repeat("─", (m.width/2)-10)),
		Bottom:       "─",
		Left:         "│",
		Right:        "│",
		TopLeft:      "╭",
		TopRight:     "╮",
		BottomLeft:   "╰",
		BottomRight:  "╯",
		MiddleLeft:   "├",
		MiddleRight:  "┤",
		Middle:       "┼",
		MiddleTop:    "┬",
		MiddleBottom: "┴",
	})
	if ae.focused == fieldName {
		// nameLabel = aeActiveLabel.Render("Name")
		nameBox = nameBox.BorderForeground(lipgloss.Color("#7fd0ff")).
			Padding(0, 1)
	}
	nameContent := ae.nameInput.View() + aeGhostStyle.Render(ae.nameSuggestion)
	out = lipgloss.JoinVertical(lipgloss.Left, out, nameBox.Render(nameContent)) //nameLabel + "  " + nameBox.Render(nameContent) + "\n\n"

	// ── Tags field ────────────────────────────────────────────────────────────
	// tagsLabel := aeLabelStyle.Render("Tags")
	tagsBox := aeFieldBox.Width(m.width/2).Align(lipgloss.Left, lipgloss.Center).Border(lipgloss.Border{
		Top:          fmt.Sprintf(" Tags %s", strings.Repeat("─", (m.width/2)-6)),
		Bottom:       "─",
		Left:         "│",
		Right:        "│",
		TopLeft:      "╭",
		TopRight:     "╮",
		BottomLeft:   "╰",
		BottomRight:  "╯",
		MiddleLeft:   "├",
		MiddleRight:  "┤",
		Middle:       "┼",
		MiddleTop:    "┬",
		MiddleBottom: "┴",
	})
	if ae.focused == fieldLabels {
		// tagsLabel = aeActiveLabel.Render("Tags")
		tagsBox = tagsBox.BorderForeground(lipgloss.Color("#7fd0ff")).
			Padding(0, 1)
	}
	tagsContent := ae.labelsInput.View() + aeGhostStyle.Render(ae.labelSuggestion)
	// out += tagsLabel + "  " + tagsBox.Render(tagsContent) + "\n\n"
	out = lipgloss.JoinVertical(lipgloss.Left, out, tagsBox.Render(tagsContent))

	// ── Hints ─────────────────────────────────────────────────────────────────
	if ae.focused == fieldName && ae.nameSuggestion != "" {
		out += aeHintStyle.Render("  tab to complete: "+ae.nameInput.Value()+ae.nameSuggestion) + "\n\n"
	} else if ae.focused == fieldLabels && ae.labelSuggestion != "" {
		out += aeHintStyle.Render("  tab to complete last tag") + "\n\n"
	} else {
		out += "\n" // keep layout stable
	}

	// ── Help bar ──────────────────────────────────────────────────────────────
	out += aeHelpStyle.Render("tab  autocomplete    enter  next / submit    esc  back")

	return tea.NewView(aeMarginStyle.Render(out))
}
