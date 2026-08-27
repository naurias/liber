package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// syncBookmarkFiles relocates the bookmark's on-disk files if its folder
// changed, then rewrites their content to match the current struct fields.
func syncBookmarkFiles(cfg Config, b *Bookmark, folderChanged bool) {
	if b.HTMLFile != "" {
		rel := b.HTMLFile
		if folderChanged {
			newRel := filepath.Join(b.Folder, filepath.Base(b.HTMLFile))
			if err := moveFile(filepath.Join(cfg.htmlDir(), rel), filepath.Join(cfg.htmlDir(), newRel)); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not move html file: %v\n", err)
			}
			rel = newRel
			b.HTMLFile = rel
		}
		if err := writeHTMLBookmark(filepath.Join(cfg.htmlDir(), rel), b); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not rewrite html file: %v\n", err)
		}
	}

	if b.MarkdownFile != "" {
		rel := b.MarkdownFile
		if folderChanged {
			newRel := filepath.Join(b.Folder, filepath.Base(b.MarkdownFile))
			if err := moveFile(filepath.Join(cfg.markdownDir(), rel), filepath.Join(cfg.markdownDir(), newRel)); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not move markdown file: %v\n", err)
			}
			rel = newRel
			b.MarkdownFile = rel
		}
		if err := writeMarkdownBookmark(filepath.Join(cfg.markdownDir(), rel), b); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not rewrite markdown file: %v\n", err)
		}
	}

	if b.ArchiveFile != "" && folderChanged {
		newRel := filepath.Join(b.Folder, filepath.Base(b.ArchiveFile))
		if err := moveFile(filepath.Join(cfg.archiveDir(), b.ArchiveFile), filepath.Join(cfg.archiveDir(), newRel)); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not move archive file: %v\n", err)
		}
		b.ArchiveFile = newRel
	}
}

func deleteBookmarkFiles(cfg Config, b *Bookmark) {
	if b.HTMLFile != "" {
		os.Remove(filepath.Join(cfg.htmlDir(), b.HTMLFile))
	}
	if b.MarkdownFile != "" {
		os.Remove(filepath.Join(cfg.markdownDir(), b.MarkdownFile))
	}
	if b.ArchiveFile != "" {
		os.Remove(filepath.Join(cfg.archiveDir(), b.ArchiveFile))
	}
}

// editBookmarkInteractive prompts for every field, pre-filled with current
// values, then persists the changes to disk (caller still must Store.Save()).
func editBookmarkInteractive(cfg Config, b *Bookmark) {
	newTitle := promptDefault("Title", b.Title)
	newDesc := promptDefault("Description", b.Description)
	newTagsLine := promptDefault("Tags (space separated)", strings.Join(b.Tags, " "))
	newFolder := sanitizeFolder(promptDefault("Folder", b.Folder))

	b.Title = newTitle
	b.Description = newDesc
	if strings.TrimSpace(newTagsLine) == "" {
		b.Tags = nil
	} else {
		b.Tags = dedupe(strings.Fields(newTagsLine))
	}
	folderChanged := newFolder != b.Folder
	b.Folder = newFolder
	b.UpdatedAt = time.Now()

	syncBookmarkFiles(cfg, b, folderChanged)
	fmt.Println("Updated.")
}

// editFlags are the flags accepted by `liber -e <id> [-t ...] [-f folder]`
// for scripted, non-interactive edits.
type editFlags struct {
	tagsSet   bool
	tags      []string
	folderSet bool
	folder    string
}

func parseEditFlags(args []string) (editFlags, error) {
	var ef editFlags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t", "--tags":
			ef.tagsSet = true
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "-") {
				ef.tags = append(ef.tags, args[i])
				i++
			}
			i--
		case "-f", "--folder":
			if i+1 >= len(args) {
				return ef, fmt.Errorf("-f requires a folder argument")
			}
			ef.folderSet = true
			ef.folder = args[i+1]
			i++
		default:
			return ef, fmt.Errorf("unknown flag for -e: %s", args[i])
		}
	}
	return ef, nil
}

func runEdit(id int, rest []string) error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	b := store.Find(id)
	if b == nil {
		return fmt.Errorf("no bookmark with id %d (see `liber -l`)", id)
	}

	if len(rest) == 0 {
		editBookmarkInteractive(cfg, b)
	} else {
		ef, err := parseEditFlags(rest)
		if err != nil {
			return err
		}
		folderChanged := false
		if ef.tagsSet {
			b.Tags = dedupe(ef.tags)
		}
		if ef.folderSet {
			newFolder := sanitizeFolder(ef.folder)
			folderChanged = newFolder != b.Folder
			b.Folder = newFolder
		}
		b.UpdatedAt = time.Now()
		syncBookmarkFiles(cfg, b, folderChanged)
		fmt.Printf("Updated [%d] %s\n", b.ID, b.Title)
	}

	return store.Save()
}
