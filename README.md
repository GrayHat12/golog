# Golog 📝

A fast, minimalistic, and beautiful Terminal User Interface (TUI) activity tracker built with Go and [Charmbracelet's Bubble Tea](https://github.com/charmbracelet/bubbletea). 

Golog allows you to seamlessly track your work activities, label them, and record breaks without leaving your terminal. All data is securely stored in a local, easy-to-parse, space-efficient JSON format.

## Features
* **Interactive TUI:** Navigate easily with arrow keys and intuitive keyboard shortcuts.
* **Smart Auto-completion:** Press `TAB` to auto-fill activity names and tags based on your historical entries.
* **Tagging System:** Append comma-separated tags to your activities for easy filtering later.
* **Portable Data:** Everything is stored in a single append-only log file in your user config directory.
* **Zero-Dependencies (System):** Compiles down to a single static binary.

## Installation

### Prerequisites
* Go 1.25.0 or higher

### Option 1: Install via `go install` (Recommended)
You can install Golog directly to your `GOPATH` using:
```bash
go install [github.com/GrayHat12/golog@latest](https://github.com/GrayHat12/golog@latest)
```

Ensure that your `$(go env GOPATH)/bin` directory is in your system's `$PATH`.

### Option 2: Build from Source

```bash
git clone [https://github.com/GrayHat12/golog.git](https://github.com/GrayHat12/golog.git)
cd golog
go build -o golog
sudo mv golog /usr/local/bin/
```

## Usage

Simply run the application from your terminal:

```bash
golog
```

### Configuration & Data Storage

By default, Golog creates a `journal.log` file inside your operating system's standard user configuration directory (e.g., `~/.config/golog/journal.log` on Linux/macOS).

You can override this location by setting an environment variable:
```bash
export GOLOG_LOG_PATH="/path/to/your/custom/journal.log"
```

### Data Format

The log file uses an ultra-compact JSON array format for individual entries to save disk space:

1. Line 1: Configuration Object (JSON)
2. Line 2+: Activity Entries: `["2026-05-20T15:31:26Z", "Activity Name", ["tag1", "tag2"]]`
   Note: A break is denoted by an empty string for the activity name.