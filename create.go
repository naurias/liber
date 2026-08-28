package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// bookmark <url> creation flags 
type CreateOptions struct {
	Interactive bool
	Markdown    bool
	Archive     bool
	Tags        []string
	Folder      string
}

func runCreate(rawURL string, opt CreateOptions) error {
	cfg, _, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	store, err := LoadStore(cfg.indexPath())
	if err != nil {
		return fmt.Errorf("loading index: %w", err)
	}

	url := normalizeURL(rawURL)
	folder := sanitizeFolder(opt.Folder)
	tags := append([]string{}, opt.Tags...)

	fmt.Printf("Fetching title for %s ...\n", url)
	title := fetchTitle(url)

	description := ""

	if opt.Interactive {
		title = promptDefault("Title", title)
		description = promptLine("Description")
		tagsLine := promptDefault("Tags (space separated)", strings.Join(tags, " "))
		tags = strings.Fields(tagsLine)
		folder = sanitizeFolder(promptDefault("Folder", folder))
	}

	if strings.TrimSpace(title) == "" {
		title = url
	}

	b := &Bookmark{
		URL:         url,
		Title:       title,
		Description: description,
		Tags:        dedupe(tags),
		Folder:      folder,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	store.Add(b) // assigns b.ID

	base := fmt.Sprintf("%04d-%s", b.ID, slugOrFallback(title, b.ID))

	htmlRel := filepath.Join(folder, base+".html")
	if err := writeHTMLBookmark(filepath.Join(cfg.htmlDir(), htmlRel), b); err != nil {
		return fmt.Errorf("writing html bookmark: %w", err)
	}
	b.HTMLFile = htmlRel

	if opt.Markdown {
		mdRel := filepath.Join(folder, base+".md")
		if err := writeMarkdownBookmark(filepath.Join(cfg.markdownDir(), mdRel), b); err != nil {
			return fmt.Errorf("writing markdown bookmark: %w", err)
		}
		b.MarkdownFile = mdRel
	}

	if opt.Archive {
		archRel := filepath.Join(folder, base+".html")
		archAbs := filepath.Join(cfg.archiveDir(), archRel)
		fmt.Println("Archiving page with single-file ...")
		if err := runSingleFile(cfg.SingleFileCmd, url, archAbs); err != nil {
			fmt.Printf("warning: archive failed: %v\n", err)
		} else {
			b.ArchiveFile = archRel
		}
	}

	// If markdown was written before the archive path was known, note it now.
	if opt.Markdown && opt.Archive && b.ArchiveFile != "" {
		_ = writeMarkdownBookmark(filepath.Join(cfg.markdownDir(), b.MarkdownFile), b)
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("saving index: %w", err)
	}

	fmt.Printf("\nSaved [%d] %s\n", b.ID, b.Title)
	fmt.Printf("  html:     %s\n", filepath.Join(cfg.htmlDir(), b.HTMLFile))
	if b.MarkdownFile != "" {
		fmt.Printf("  markdown: %s\n", filepath.Join(cfg.markdownDir(), b.MarkdownFile))
	}
	if b.ArchiveFile != "" {
		fmt.Printf("  archive:  %s\n", filepath.Join(cfg.archiveDir(), b.ArchiveFile))
	}
	return nil
}

func slugOrFallback(title string, id int) string {
	s := slugify(title)
	if s == "" {
		return fmt.Sprintf("bookmark-%d", id)
	}
	return s
}
