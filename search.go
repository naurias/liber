package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runSearch(fields SearchFields) error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	if len(store.Bookmarks) == 0 {
		fmt.Println("No bookmarks yet. Add one with: liber <url>")
		return nil
	}

	if fzfAvailable() {
		if err := runSearchFzf(cfg, store, fields); err != nil {
			fmt.Printf("(fzf picker failed: %v -- falling back to plain search)\n", err)
		} else {
			return nil
		}
	} else {
		fmt.Println("(tip: install fzf for a fuzzy picker here -- falling back to plain search)")
	}
	return runSearchPrompt(cfg, store, fields)
}

// runSearchLegacy skips the fzf check entirely, for `liber -sl` (and its
// field-restricted variants like `-sld`) -- useful if you have fzf
// installed but want the plain prompt anyway.
func runSearchLegacy(fields SearchFields) error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	if len(store.Bookmarks) == 0 {
		fmt.Println("No bookmarks yet. Add one with: liber <url>")
		return nil
	}
	return runSearchPrompt(cfg, store, fields)
}

// runSearchFzf pipes the bookmark list to fzf for fuzzy selection, then
// hands the pick off to the same open/edit/delete action menu used by the
// plain-prompt path. It loops so edits/deletes are reflected next time the
// picker opens. A genuine fzf failure (as opposed to the user cancelling)
// is returned to the caller so it can fall back to the plain prompt.
func runSearchFzf(cfg Config, store *Store, fields SearchFields) error {
	for {
		all := store.All()
		if len(all) == 0 {
			fmt.Println("No bookmarks left.")
			return nil
		}
		id, ok, err := pickWithFzf(all, fields)
		if err != nil {
			return err
		}
		if !ok {
			return nil // user cancelled
		}
		b := store.Find(id)
		if b == nil {
			continue
		}
		if actionMenu(cfg, store, b) == actionQuit {
			return nil
		}
	}
}

// runSearchPrompt is the dependency-free fallback: type a query, get a
// numbered list, type an id to act on it.
func runSearchPrompt(cfg Config, store *Store, fields SearchFields) error {
	label := fmt.Sprintf("Search %s (empty = all, 'q' to quit)", fields.Label())
	for {
		q := promptLine(label)
		if q == "q" {
			return nil
		}
		results := store.Search(q, fields)
		if len(results) == 0 {
			fmt.Println("No matches.")
			continue
		}
		printResults(results)

		for {
			sel := promptLine("id to open/edit ('s' new search, 'q' quit)")
			if sel == "q" {
				return nil
			}
			if sel == "s" || sel == "" {
				break
			}
			id, err := strconv.Atoi(sel)
			if err != nil {
				fmt.Println("Not a valid id.")
				continue
			}
			b := store.Find(id)
			if b == nil {
				fmt.Println("No bookmark with that id.")
				continue
			}
			done := actionMenu(cfg, store, b)
			if done == actionQuit {
				return nil
			}
		}
	}
}

type actionResult int

const (
	actionContinue actionResult = iota
	actionQuit
)

func actionMenu(cfg Config, store *Store, b *Bookmark) actionResult {
	for {
		fmt.Printf("\n[%d] %s\n    %s\n", b.ID, b.Title, b.URL)
		opts := "(o)pen"
		if b.MarkdownFile != "" {
			opts += "  (m)arkdown"
		}
		if b.ArchiveFile != "" {
			opts += "  (a)rchive"
		}
		opts += "  (e)dit  (d)elete  (b)ack  (q)uit"
		choice := strings.ToLower(promptLine(opts))
		switch choice {
		case "o":
			if err := openURL(cfg, b.URL); err != nil {
				fmt.Println("Could not open browser:", err)
			} else {
				now := time.Now()
				b.LastOpenedAt = &now
				b.OpenCount++
				if err := store.Save(); err != nil {
					fmt.Println("Could not save index:", err)
				}
			}
		case "m":
			if b.MarkdownFile == "" {
				fmt.Println("No markdown copy saved for this bookmark.")
				continue
			}
			path := filepath.Join(cfg.markdownDir(), b.MarkdownFile)
			if err := openInEditor(cfg, path); err != nil {
				fmt.Println("Could not open markdown file:", err)
			}
		case "a":
			if b.ArchiveFile == "" {
				fmt.Println("No archive saved for this bookmark.")
				continue
			}
			path := filepath.Join(cfg.archiveDir(), b.ArchiveFile)
			if err := openURL(cfg, path); err != nil {
				fmt.Println("Could not open archive:", err)
			}
		case "e":
			editBookmarkInteractive(cfg, b)
			if err := store.Save(); err != nil {
				fmt.Println("Could not save index:", err)
			}
		case "d":
			if confirm(fmt.Sprintf("Delete '%s'?", b.Title), false) {
				deleteBookmarkFiles(cfg, b)
				store.Delete(b.ID)
				if err := store.Save(); err != nil {
					fmt.Println("Could not save index:", err)
				}
				fmt.Println("Deleted.")
				return actionContinue
			}
		case "b", "":
			return actionContinue
		case "q":
			return actionQuit
		default:
			fmt.Println("Unknown option.")
		}
	}
}

func printResults(list []*Bookmark) {
	fmt.Println()
	for _, b := range list {
		folder := b.Folder
		if folder == "" {
			folder = "/"
		}
		tags := ""
		if len(b.Tags) > 0 {
			tags = " #" + strings.Join(b.Tags, " #")
		}
		fmt.Printf("[%d] %s%s\n     %s\n     folder:%s%s\n", b.ID, b.Title, badgeSuffix(b), b.URL, folder, tags)
	}
	fmt.Println()
}

// badgeSuffix renders " [md]", " [arc]", " [md,arc]", or "" depending on
// which of a bookmark's markdown/archive copies exist.
func badgeSuffix(b *Bookmark) string {
	var parts []string
	if b.MarkdownFile != "" {
		parts = append(parts, "md")
	}
	if b.ArchiveFile != "" {
		parts = append(parts, "arc")
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, ",") + "]"
}

func runList() error {
	_, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	all := store.All()
	if len(all) == 0 {
		fmt.Println("No bookmarks yet. Add one with: liber <url>")
		return nil
	}
	printResults(all)
	return nil
}
