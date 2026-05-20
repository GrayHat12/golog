package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/GrayHat12/golog/lib"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

type sessionState int

const (
	stateSelectType sessionState = iota
	stateInputName
	stateInputLabels
	stateDone
)

type model struct {
	app         *lib.Application
	state       sessionState
	choiceIdx   int // 0 for Work, 1 for Break
	nameInput   string
	labelsInput string
	err         error // Track execution errors
}

func initialModel(app *lib.Application) model {
	return model{
		app:       app,
		state:     stateSelectType,
		choiceIdx: 0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.state {
		case stateSelectType:
			switch msg.String() {
			case "up", "k":
				m.choiceIdx = 0
			case "down", "j":
				m.choiceIdx = 1
			case "enter":
				if m.choiceIdx == 1 { // Break chosen
					m.err = m.app.AddEntry(nil)
					m.state = stateDone
					return m, tea.Quit
				}
				m.state = stateInputName
			}

		case stateInputName:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.nameInput) != "" {
					m.state = stateInputLabels
				}
			case "backspace":
				if len(m.nameInput) > 0 {
					m.nameInput = m.nameInput[:len(m.nameInput)-1]
				}
			case "tab":
				// Autocomplete action
				names, _ := m.app.GetNameSuggestions()
				if sugg := getSuggestion(m.nameInput, names); sugg != "" {
					m.nameInput = sugg
				}
			case "space":
				m.nameInput += " "
			default:
				// Avoid catching control keys as typing characters
				if len(msg.String()) == 1 {
					m.nameInput += msg.String()
				}
			}

		case stateInputLabels:
			switch msg.String() {
			case "enter":
				// Build list of tags from string buffer
				rawTags := strings.Split(m.labelsInput, ",")
				var cleanTags []string
				for _, tag := range rawTags {
					trimmed := strings.TrimSpace(tag)
					if trimmed != "" {
						cleanTags = append(cleanTags, trimmed)
					}
				}

				entry := &lib.Entry{
					Timestamp: time.Now().UTC(),
					Name:      m.nameInput,
					Labels:    cleanTags,
				}
				m.err = m.app.AddEntry(entry)
				m.state = stateDone
				return m, tea.Quit

			case "backspace":
				if len(m.labelsInput) > 0 {
					m.labelsInput = m.labelsInput[:len(m.labelsInput)-1]
				}
			case "tab":
				labels, _ := m.app.GetLabelSuggestions()
				m.labelsInput = autocompleteLastTag(m.labelsInput, labels)
			default:
				if len(msg.String()) == 1 {
					m.labelsInput += msg.String()
				}
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var s strings.Builder
	s.WriteString("\n--- Activity Tracker ---\n\n")

	switch m.state {
	case stateSelectType:
		s.WriteString("Select the type of entry:\n")
		options := []string{"Work Activity", "Take a Break"}
		for i, option := range options {
			cursor := " "
			if m.choiceIdx == i {
				cursor = ">"
				s.WriteString(fmt.Sprintf("%s \033[1;36m%s\033[0m\n", cursor, option))
			} else {
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, option))
			}
		}
		s.WriteString("\n[Use Up/Down arrows to toggle • Enter to select]")

	case stateInputName:
		s.WriteString(fmt.Sprintf("Activity Name: \033[1;32m%s\033[0m█\n", m.nameInput))
		names, _ := m.app.GetNameSuggestions()
		if sugg := getSuggestion(m.nameInput, names); sugg != "" {
			s.WriteString(fmt.Sprintf("\033[90mSuggestion: %s [Press TAB to auto-fill]\033[0m\n", sugg))
		}
		s.WriteString("\n[Type entry name • Enter to continue]")

	case stateInputLabels:
		s.WriteString(fmt.Sprintf("Activity Name: \033[32m%s\033[0m\n", m.nameInput))
		s.WriteString(fmt.Sprintf("Labels (comma separated): \033[1;32m%s\033[0m█\n", m.labelsInput))
		labels, _ := m.app.GetLabelSuggestions()
		if sugg := getTagSuggestion(m.labelsInput, labels); sugg != "" {
			s.WriteString(fmt.Sprintf("\033[90mSuggestion: %s [Press TAB to auto-fill]\033[0m\n", sugg))
		}
		s.WriteString("\n[Enter tags • Enter to save and submit]")

	case stateDone:
		if m.err != nil {
			s.WriteString(fmt.Sprintf("\033[1;31mError saving entry: %v\033[0m\n", m.err))
		} else {
			s.WriteString("\033[1;32m✓ Entry successfully appended to application log!\033[0m\n")
		}
	}

	s.WriteString("\n\033[90mPress or Ctrl+C to abort\033[0m\n")

	return tea.NewView(s.String())
}

func getSuggestion(input string, dict []string) string {
	if input == "" || len(dict) == 0 {
		return ""
	}

	var bestMatch string
	bestScore := -1
	queryRunes := []rune(input)

	slab := util.MakeSlab(100*1024, 2048)

	for _, item := range dict {
		inputChars := util.ToChars([]byte(item))

		result, _ := algo.FuzzyMatchV2(false, true, true, &inputChars, queryRunes, false, slab)

		if result.Score > bestScore {
			bestScore = result.Score
			bestMatch = item
		}
	}

	if bestScore <= 0 {
		return ""
	}

	return bestMatch
}

func getTagSuggestion(input string, dict []string) string {
	parts := strings.Split(input, ",")
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	if len(lastPart) == 0 {
		return ""
	}
	return getSuggestion(lastPart, dict)
}

func autocompleteLastTag(input string, dict []string) string {
	parts := strings.Split(input, ",")
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	sugg := getSuggestion(lastPart, dict)
	if sugg == "" {
		return input
	}
	parts[len(parts)-1] = " " + sugg
	return strings.Join(parts, ",")
}

func getApplicationName() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Path != "" {
		return filepath.Base(info.Main.Path)
	}
	return "golog"
}

func GetLogPath() (string, error) {
	application_name := getApplicationName()
	if customPath := os.Getenv(fmt.Sprintf("%s_LOG_PATH", strings.ToUpper(application_name))); customPath != "" {
		return filepath.Clean(customPath), nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	defaultPath := filepath.Join(configDir, fmt.Sprintf("%s", application_name), "journal.log")
	dir := filepath.Dir(defaultPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create log directory %s: %v", dir, err)
	}
	return defaultPath, nil
}

func main() {
	logPath, err := GetLogPath()
	fmt.Printf("using %s\n", logPath)
	if err != nil {
		log.Fatalf("Error determining log path: %v", err)
	}
	app, err := lib.NewApplication(logPath)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
	defer app.Close()
	app.Initialise()
	_, _ = app.Summarise()
	p := tea.NewProgram(initialModel(app))

	_, err = p.Run()
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
