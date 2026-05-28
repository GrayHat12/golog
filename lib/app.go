package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

type Application struct {
	file             *os.File
	Config           *Config
	lock             sync.RWMutex
	suggestions      map[string]map[string]struct{}
	currentlyWorking *Entry
}

func NewApplication(path string) (*Application, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &Application{
		file:        file,
		lock:        sync.RWMutex{},
		suggestions: make(map[string]map[string]struct{}),
	}, err
}

func (app *Application) Initialise() (*Config, error) {
	app.lock.Lock()
	defer app.lock.Unlock()
	_, err := app.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	var config Config

	reader := bufio.NewReader(app.file)

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}

	line = strings.TrimSpace(line)
	if line != "" {
		if err := json.Unmarshal([]byte(line), &config); err != nil {
			return &config, fmt.Errorf("failed to parse config line: %w", err)
		}
	}

	if err == io.EOF {
		// incase of a new file, insert a default config
		jsonData, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}
		_, err = app.file.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, err
		}
		_, err = app.file.WriteString(string(jsonData) + "\n")
		if err != nil {
			return nil, err
		}
	}

	app.Config = &config

	return &config, nil
}

func (app *Application) Summarise() (*Summary, error) {
	app.lock.Lock()
	defer app.lock.Unlock()
	_, err := app.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(app.file)
	isFirstLine := true

	summary := Summary{}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line != "" {
			if isFirstLine {
				isFirstLine = false
				continue
			} else {
				var entry Entry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					// You can choose to break, log, or skip here
					fmt.Printf("Skipping malformed line: %s (Error: %v)\n", line, err)
					continue
				}
				if entry.IsBreak() {
					summary.AddBreak(entry.Timestamp)
				} else {
					summary.AddActivity(entry)
				}
			}
		}

		if err == io.EOF {
			break
		}
	}

	return &summary, nil
}

func (app *Application) appendEntry(entry Entry) error {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	app.lock.Lock()
	defer app.lock.Unlock()
	_, err = app.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	_, err = app.file.WriteString(string(jsonData) + "\n")
	if err == nil {
		app.currentlyWorking = &entry
	}
	return err
}

func (app *Application) AddEntry(entry *Entry) error {
	if entry == nil {
		currentWork, err := app.CurrentlyWorkingOn()
		if err == nil && currentWork.IsBreak() {
			return nil
		}
		// add break
		return app.appendEntry(Entry{
			Timestamp: time.Now().UTC(),
			Name:      "", // empty string denotes a break
		})
	} else {
		// add entry
		err := app.appendEntry(*entry)
		if err != nil {
			return err
		} else {
			// add entry for suggestion
			if _, exists := app.suggestions[entry.Name]; !exists {
				app.suggestions[entry.Name] = make(map[string]struct{})
			}
			labels := map[string]struct{}{}
			for _, tag := range entry.Labels {
				labels[tag] = struct{}{}
			}
			app.suggestions[entry.Name] = labels
		}
		return err
	}
}

func (app *Application) CurrentlyWorkingOn() (*Entry, error) {
	if app.currentlyWorking != nil {
		return app.currentlyWorking, nil
	}

	app.lock.Lock()
	defer app.lock.Unlock()
	_, err := app.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(app.file)
	isFirstLine := true
	var entry Entry

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line != "" {
			if isFirstLine {
				isFirstLine = false
				continue
			} else {
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					// You can choose to break, log, or skip here
					fmt.Printf("Skipping malformed line: %s (Error: %v)\n", line, err)
					continue
				}
			}
		}

		if err == io.EOF {
			break
		}
	}

	app.currentlyWorking = &entry

	return app.currentlyWorking, nil
}

func (app *Application) Close() error {
	return app.file.Close()
}

func (app *Application) PopulateSuggestions() error {
	summary, err := app.Summarise()
	if err != nil {
		return err
	}
	for _, activity := range summary.Activities {
		if _, ok := app.suggestions[activity.Name]; !ok {
			app.suggestions[activity.Name] = map[string]struct{}{}
		}
		for _, tag := range activity.Tags {
			app.suggestions[activity.Name][tag] = struct{}{}
		}
	}
	return nil
}

func (app *Application) GetNameSuggestions() ([]string, error) {
	if len(app.suggestions) <= 0 {
		err := app.PopulateSuggestions()
		if err != nil {
			return []string{}, err
		}
	}
	return slices.Collect(maps.Keys(app.suggestions)), nil
}

func (app *Application) GetLabelSuggestions() ([]string, error) {
	if len(app.suggestions) <= 0 {
		err := app.PopulateSuggestions()
		if err != nil {
			return []string{}, err
		}
	}
	suggestions := []string{}
	for _, labels := range app.suggestions {
		suggestions = slices.AppendSeq(suggestions, maps.Keys(labels))
	}
	return suggestions, nil
}

func (app *Application) PrefilLabel(name *string) ([]string, error) {
	if len(app.suggestions) <= 0 {
		err := app.PopulateSuggestions()
		if err != nil {
			return []string{}, err
		}
	}
	if name == nil {
		return []string{}, nil
	}
	labels, ok := app.suggestions[*name]
	if ok {
		return slices.Collect(maps.Keys(labels)), nil
	} else {
		return []string{}, nil
	}
}
