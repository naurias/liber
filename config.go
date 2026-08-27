package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds liber's user-editable settings. Stored as JSON under the
// standard user config directory (e.g. ~/.config/liber/config.json).
type Config struct {
	// BaseDir is the root of the bookmark collection. html/, markdown/,
	// archive/ and the .liber index live under here unless overridden below.
	BaseDir string `json:"base_dir"`

	HTMLDir     string `json:"html_dir,omitempty"`
	MarkdownDir string `json:"markdown_dir,omitempty"`
	ArchiveDir  string `json:"archive_dir,omitempty"`

	// SingleFileCmd is the executable used to produce full-page archives.
	// See https://github.com/gildas-lormeau/single-file-cli
	SingleFileCmd string `json:"singlefile_cmd,omitempty"`

	// BrowserCmd overrides the command used to open URLs from `liber -s`.
	// Leave empty to auto-detect (xdg-open / open / start).
	BrowserCmd string `json:"browser_cmd,omitempty"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "liber", "config.json"), nil
}

func defaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		BaseDir:       filepath.Join(home, "Bookmarks"),
		SingleFileCmd: "single-file",
	}
}

// LoadConfig reads the config file, creating a default one on first run.
func LoadConfig() (Config, string, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, "", err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		if werr := SaveConfig(cfg); werr != nil {
			// Not fatal: fall back to in-memory defaults.
			fmt.Fprintf(os.Stderr, "warning: could not write default config: %v\n", werr)
		}
		return cfg, path, nil
	} else if err != nil {
		return Config{}, path, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, path, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.SingleFileCmd == "" {
		cfg.SingleFileCmd = "single-file"
	}
	if cfg.BaseDir == "" {
		cfg.BaseDir = defaultConfig().BaseDir
	}
	return cfg, path, nil
}

func SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (c Config) htmlDir() string {
	if c.HTMLDir != "" {
		return expandTilde(c.HTMLDir)
	}
	return filepath.Join(expandTilde(c.BaseDir), "html")
}

func (c Config) markdownDir() string {
	if c.MarkdownDir != "" {
		return expandTilde(c.MarkdownDir)
	}
	return filepath.Join(expandTilde(c.BaseDir), "markdown")
}

func (c Config) archiveDir() string {
	if c.ArchiveDir != "" {
		return expandTilde(c.ArchiveDir)
	}
	return filepath.Join(expandTilde(c.BaseDir), "archive")
}

func (c Config) indexPath() string {
	return filepath.Join(expandTilde(c.BaseDir), ".liber", "index.json")
}
