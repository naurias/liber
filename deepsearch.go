package main

import (
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var scriptStyleRe = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)

// maxArchiveScanBytes caps how much of one archive file deep search reads.
const maxArchiveScanBytes = 5 * 1024 * 1024

// extractArchiveText is a best-effort HTML-to-text conversion; see dev-docs.md#deep-search.
func extractArchiveText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxArchiveScanBytes))
	if err != nil {
		return "", err
	}
	s := string(data)
	s = scriptStyleRe.ReplaceAllString(s, " ")
	s = stripTagRe.ReplaceAllString(s, " ")
	return html.UnescapeString(s), nil
}

// filterDeep narrows list to metadata-or-archive-content matches; see dev-docs.md#deep-search.
func filterDeep(cfg Config, list []*Bookmark, query string, fields SearchFields) []*Bookmark {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []*Bookmark
	for _, b := range list {
		if bookmarkMatches(b, q, fields) {
			out = append(out, b)
			continue
		}
		if b.ArchiveFile != "" {
			text, err := extractArchiveText(filepath.Join(cfg.archiveDir(), b.ArchiveFile))
			if err == nil && strings.Contains(strings.ToLower(text), q) {
				out = append(out, b)
			}
		}
	}
	return out
}
