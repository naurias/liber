package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// runPreview prints a detailed, labeled view of one bookmark. It's not a
// user-facing command -- fzf invokes `liber __preview <id>` per highlighted
// row to fill its preview pane (see fzf.go). It never returns an error to
// its caller: any problem just renders as a friendly placeholder, since a
// failed preview shouldn't interrupt the picker.
func runPreview(id int) error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		fmt.Println("(could not load bookmarks)")
		return nil
	}
	b := store.Find(id)
	if b == nil {
		fmt.Println("(no details available)")
		return nil
	}

	folder := b.Folder
	if folder == "" {
		folder = "/"
	}
	tags := "(none)"
	if len(b.Tags) > 0 {
		tags = "#" + strings.Join(b.Tags, " #")
	}
	desc := b.Description
	if desc == "" {
		desc = "(none)"
	}

	fmt.Printf("\x1b[1;36mTitle\x1b[0m\n%s\n\n", b.Title)
	fmt.Printf("\x1b[1;34mURL\x1b[0m\n%s\n\n", b.URL)
	fmt.Printf("\x1b[1;33mTags\x1b[0m\n%s\n\n", tags)
	fmt.Printf("\x1b[1;32mFolder\x1b[0m\n%s\n\n", folder)
	fmt.Printf("\x1b[1mDescription\x1b[0m\n%s\n", desc)

	fmt.Printf("\n\x1b[2mSaved %s \u00b7 id %d\x1b[0m\n", b.CreatedAt.Format("Jan 2, 2006"), b.ID)
	if b.MarkdownFile != "" {
		fmt.Printf("\x1b[2mmarkdown: %s\x1b[0m\n", filepath.Join(cfg.markdownDir(), b.MarkdownFile))
	}
	if b.ArchiveFile != "" {
		fmt.Printf("\x1b[2marchive:  %s\x1b[0m\n", filepath.Join(cfg.archiveDir(), b.ArchiveFile))
	}
	return nil
}
