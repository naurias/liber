package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// expandTilde expands a leading ~ or ~/ to the user's home directory.
func expandTilde(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

var slugInvalid = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts s into a lowercase, hyphen-separated slug ("" if nothing usable remains).
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugInvalid.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
		s = strings.Trim(s, "-")
	}
	return s
}

// sanitizeFolder strips ".." traversal, leading slashes, and empty segments.
func sanitizeFolder(f string) string {
	f = strings.TrimSpace(f)
	f = strings.ReplaceAll(f, "\\", "/")
	parts := strings.Split(f, "/")
	var clean []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "." || p == ".." {
			continue
		}
		clean = append(clean, p)
	}
	return strings.Join(clean, "/")
}

// dedupe removes empty/duplicate (case-insensitive) entries, preserving order.
func dedupe(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		key := strings.ToLower(it)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, it)
	}
	return out
}

// normalizeURL adds an https:// scheme if none was given.
func normalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	return u
}

// moveFile is a no-op if src doesn't exist or src == dst.
func moveFile(src, dst string) error {
	if src == dst {
		return nil
	}
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// displayFolder returns f, or "/" if it's the root (empty string).
func displayFolder(f string) string {
	if f == "" {
		return "/"
	}
	return f
}

// fileExists reports whether path exists and is readable via os.Stat.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func atoiOr(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return n
}
