package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/GrayHat12/golog/lib"
	"github.com/GrayHat12/golog/tui"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

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

func parseLabels(labels string) []string {
	tags := []string{}
	for _, tag := range strings.Split(labels, ",") {
		tags = append(tags, strings.TrimSpace(tag))
	}
	return tags
}

func main() {
	logPath, err := GetLogPath()
	// fmt.Printf("using %s\n", logPath)
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

	err = tui.Start(tui.Config{
		JournalPath: logPath,
		OnAddEntry: func(e tui.EntryInput) error {
			labels := parseLabels(e.Labels) // split & trim yourself
			return app.AddEntry(&lib.Entry{Name: e.Name, Labels: labels, Timestamp: time.Now().UTC()})
		},
		OnAddBreak: func() error { return app.AddEntry(nil) },
		CurrentName: func() string {
			entry, err := app.CurrentlyWorkingOn()
			if err != nil {
				return fmt.Sprintf("error %v", err)
			} else {
				return entry.Name
			}
		},
		CurrentStart: func() time.Time {
			entry, err := app.CurrentlyWorkingOn()
			if err != nil {
				return time.Now()
			} else {
				return entry.Timestamp
			}
		},

		NameSuggestions: func() []string {
			suggestions, _ := app.GetNameSuggestions()
			return suggestions
		},
		LabelSuggestions: func() []string {
			suggestions, _ := app.GetLabelSuggestions()
			return suggestions
		},

		// your existing functions drop in directly:
		GetNameSuggestion:   getSuggestion,
		AutocompleteName:    getSuggestion,
		GetTagSuggestion:    getTagSuggestion,
		AutocompleteLastTag: autocompleteLastTag,
		GetSummary: func() *lib.Summary {
			summary, err := app.Summarise()
			if err != nil {
				return nil
			}
			return summary
		},
	})
}
