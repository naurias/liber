package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Bookmark is a single saved link and its metadata. Files on disk
// (HTMLFile / MarkdownFile / ArchiveFile) are stored as paths relative to
// their respective directory (cfg.htmlDir(), etc).
type Bookmark struct {
	ID          int       `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Folder      string    `json:"folder,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	HTMLFile     string `json:"html_file,omitempty"`
	MarkdownFile string `json:"markdown_file,omitempty"`
	ArchiveFile  string `json:"archive_file,omitempty"`
}

// Store is the full bookmark index, persisted as a single JSON file.
type Store struct {
	NextID    int         `json:"next_id"`
	Bookmarks []*Bookmark `json:"bookmarks"`

	path string // not persisted
}

func LoadStore(path string) (*Store, error) {
	s := &Store{NextID: 1, path: path}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	} else if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	s.path = path
	if s.NextID == 0 {
		s.NextID = 1
	}
	return s, nil
}

// Save writes the index atomically (write to temp file, then rename).
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Add assigns the next ID to b and appends it to the store.
func (s *Store) Add(b *Bookmark) {
	b.ID = s.NextID
	s.NextID++
	s.Bookmarks = append(s.Bookmarks, b)
}

func (s *Store) Find(id int) *Bookmark {
	for _, b := range s.Bookmarks {
		if b.ID == id {
			return b
		}
	}
	return nil
}

func (s *Store) Delete(id int) bool {
	for i, b := range s.Bookmarks {
		if b.ID == id {
			s.Bookmarks = append(s.Bookmarks[:i], s.Bookmarks[i+1:]...)
			return true
		}
	}
	return false
}

// Search does a case-insensitive substring match across title, URL,
// folder, description and tags. An empty query returns everything.
func (s *Store) Search(query string) []*Bookmark {
	query = strings.ToLower(strings.TrimSpace(query))
	var results []*Bookmark
	for _, b := range s.Bookmarks {
		if query == "" || bookmarkMatches(b, query) {
			results = append(results, b)
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results
}

func (s *Store) All() []*Bookmark {
	out := append([]*Bookmark{}, s.Bookmarks...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// loadCfgAndStore is the common "load config, then load the index it points
// at" pairing used by nearly every command.
func loadCfgAndStore() (Config, *Store, error) {
	cfg, _, err := LoadConfig()
	if err != nil {
		return Config{}, nil, fmt.Errorf("loading config: %w", err)
	}
	store, err := LoadStore(cfg.indexPath())
	if err != nil {
		return Config{}, nil, fmt.Errorf("loading index: %w", err)
	}
	return cfg, store, nil
}

func bookmarkMatches(b *Bookmark, q string) bool {
	if strings.Contains(strings.ToLower(b.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(b.URL), q) {
		return true
	}
	if strings.Contains(strings.ToLower(b.Folder), q) {
		return true
	}
	if strings.Contains(strings.ToLower(b.Description), q) {
		return true
	}
	for _, t := range b.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}
