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
	nameSuggestion   map[string]struct{}
	labelSuggestions map[string]struct{}
}

func NewApplication(path string) (*Application, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &Application{file: file, lock: sync.RWMutex{}, nameSuggestion: make(map[string]struct{}), labelSuggestions: make(map[string]struct{})}, err
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
		app.nameSuggestion[entry.Name] = struct{}{}
		for _, label := range entry.Labels {
			app.labelSuggestions[label] = struct{}{}
		}
		return app.appendEntry(*entry)
	}
}

func (app *Application) CurrentlyWorkingOn() (*Entry, error) {
	app.lock.Lock()
	defer app.lock.Unlock()

	cursor := int64(-1)
	line := ""

	stat, err := app.file.Stat()
	if err != nil {
		return nil, err
	}
	filesize := stat.Size()

	for {
		_, err := app.file.Seek(cursor, io.SeekEnd)
		if err != nil {
			return nil, err
		}

		cursor -= 1

		char := make([]byte, 1)
		_, err = app.file.Read(char)
		if err != nil {
			return nil, err
		}

		if cursor < -2 && (char[0] == 10 || char[0] == 13) { // stop if we find a line
			break
		}

		line = fmt.Sprintf("%s%s", string(char), line)
		if cursor <= -filesize { // stop if we are at the begining
			break
		}
	}

	var entry Entry

	if err = json.Unmarshal([]byte(line), &entry); err != nil {
		// You can choose to break, log, or skip here
		// fmt.Printf("Got malformed line: %s (Error: %v)\n", line, err)
		return nil, err
	}

	return &entry, nil
}

func (app *Application) Close() error {
	return app.file.Close()
}

func (app *Application) GetNameSuggestions() ([]string, error) {
	if len(app.nameSuggestion) > 0 {
		return slices.Collect(maps.Keys(app.nameSuggestion)), nil
	}
	summary, err := app.Summarise()
	if err != nil {
		return []string{}, err
	}
	names := map[string]struct{}{}
	for _, activity := range summary.Activities {
		names[activity.Name] = struct{}{}
	}

	app.nameSuggestion = names

	return slices.Collect(maps.Keys(names)), nil
}

func (app *Application) GetLabelSuggestions() ([]string, error) {
	if len(app.labelSuggestions) > 0 {
		return slices.Collect(maps.Keys(app.labelSuggestions)), nil
	}
	summary, err := app.Summarise()
	if err != nil {
		return []string{}, err
	}
	labels := map[string]struct{}{}
	for _, activity := range summary.Activities {
		for _, label := range activity.Tags {
			labels[label] = struct{}{}
		}
	}
	app.labelSuggestions = labels
	return slices.Collect(maps.Keys(labels)), nil
}
