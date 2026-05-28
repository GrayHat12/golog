// Package tui provides a BubbleTea v2 terminal UI for a time-tracker application.
//
// Usage:
//
//	err := tui.Start(tui.Config{
//	    OnAddEntry: func(e tui.EntryInput) error {
//	        return app.AddEntry(&Entry{
//	            Name:   e.Name,
//	            Labels: strings.Split(e.Labels, ","),
//	        })
//	    },
//	    OnAddBreak:       func() error { return app.AddEntry(nil) },
//	    CurrentName:      func() string { return app.CurrentEntry().Name },
//	    CurrentStart:     func() time.Time { return app.CurrentEntry().Timestamp },
//	    NameSuggestions:  func() []string { return app.NameSuggestionList() },
//	    LabelSuggestions: func() []string { return app.LabelSuggestionList() },
//	    GetNameSuggestion:  getSuggestion,
//	    AutocompleteName:   getSuggestion, // or a wrapper that returns the full completion
//	    GetTagSuggestion:   getTagSuggestion,
//	    AutocompleteLastTag: autocompleteLastTag,
//	})
package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/GrayHat12/golog/lib"
	// tea "github.com/charmbracelet/bubbletea/v2"
)

// EntryInput holds the raw values captured from the add-entry form.
// The caller is responsible for parsing Labels (comma-separated) into their
// application's Entry type.
type EntryInput struct {
	Name   string // entry name
	Labels string // raw comma-separated label string, e.g. "golang, backend, api"
}

// Config wires the TUI to your application logic.
// All function fields are optional; nil values are safely ignored.
type Config struct {
	JournalPath string
	// OnAddEntry is called when the user submits the add-entry form.
	// Return a non-nil error to surface it on the home screen.
	OnAddEntry func(e EntryInput) error

	// OnAddBreak is called when the user presses 'b' on the home screen.
	// Return a non-nil error to surface it on the home screen.
	OnAddBreak func() error

	// CurrentName returns the display name of the currently tracked entry.
	// Return an empty string to indicate a break / idle state.
	CurrentName func() string

	// CurrentStart returns the start time of the currently tracked entry.
	// Return a zero time.Time to indicate nothing is running.
	CurrentStart func() time.Time

	// ── Autocomplete ──────────────────────────────────────────────────────────
	// Plug your getSuggestion / getTagSuggestion / autocompleteLastTag helpers
	// directly into these fields.

	// GetNameSuggestion(input, dict) → full suggested name, or "" if no match.
	// Maps to your getSuggestion function.
	GetNameSuggestion func(input string, dict []string) string

	// AutocompleteName(input, dict) → input with the name field autocompleted.
	// Often just a thin wrapper around getSuggestion that returns the suggestion
	// directly; override if you need different behaviour (e.g. preserve casing).
	AutocompleteName func(input string, dict []string) string

	// GetTagSuggestion(input, dict) → suggested completion for the last tag, or "".
	// Maps directly to your getTagSuggestion function.
	GetTagSuggestion func(input string, dict []string) string

	// AutocompleteLastTag(input, dict) → full input string with the last tag completed.
	// Maps directly to your autocompleteLastTag function.
	AutocompleteLastTag func(input string, dict []string) string

	// ── Suggestion dictionaries ───────────────────────────────────────────────
	// These are called on every keystroke, so return a snapshot / copy if needed.

	// NameSuggestions returns the current pool of known entry names.
	NameSuggestions func() []string

	// LabelSuggestions returns the current pool of known labels/tags.
	LabelSuggestions func() []string

	// Prefil Label returns the current pool of known labels/tags.
	PrefilLabels func(name *string) []string

	GetSummary func() *lib.Summary
}

// Start launches the TUI in alt-screen mode and blocks until the user quits
// (Esc on the home screen or Ctrl+C anywhere).
func Start(cfg Config) error {
	m := newModel(cfg)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
