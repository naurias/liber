package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runSearch(fields SearchFields, deep bool) error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	if len(store.Bookmarks) == 0 {
		fmt.Println("No bookmarks yet. Add one with: liber <url>")
		return nil
	}

	if deep {
		list, ok := promptDeepQuery(cfg, store, fields)
		if !ok {
			return nil
		}
		if fzfAvailable() {
			if err := runSearchFzfList(cfg, store, fields, list); err != nil {
				fmt.Printf("(fzf picker failed: %v -- falling back to plain list)\n", err)
			} else {
				return nil
			}
		} else {
			fmt.Println("(tip: install fzf for a fuzzy picker here -- falling back to plain list)")
		}
		return runPlainListLoop(cfg, store, list)
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

// runSearchLegacy skips the fzf check (used by `liber -sl` and its field-restricted variants).
func runSearchLegacy(fields SearchFields, deep bool) error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	if len(store.Bookmarks) == 0 {
		fmt.Println("No bookmarks yet. Add one with: liber <url>")
		return nil
	}
	if deep {
		list, ok := promptDeepQuery(cfg, store, fields)
		if !ok {
			return nil
		}
		return runPlainListLoop(cfg, store, list)
	}
	return runSearchPrompt(cfg, store, fields)
}

// promptDeepQuery asks once for a query and returns the filtered list; see dev-docs.md#deep-search.
func promptDeepQuery(cfg Config, store *Store, fields SearchFields) ([]*Bookmark, bool) {
	q := promptLine(fmt.Sprintf("Deep search %s + archive content (empty = all)", fields.Label()))
	if strings.TrimSpace(q) == "" {
		fmt.Println("No query given -- showing everything.")
		return store.All(), true
	}
	fmt.Println("Searching archives, this may take a moment...")
	list := filterDeep(cfg, store.All(), q, fields)
	if len(list) == 0 {
		fmt.Println("No matches.")
		return nil, false
	}
	return list, true
}

// runSearchFzf loops the fzf picker (fresh bookmark list each time) into the action menu.
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

// runSearchFzfList is runSearchFzf over a fixed list (deep search's one-time snapshot).
func runSearchFzfList(cfg Config, store *Store, fields SearchFields, list []*Bookmark) error {
	for {
		if len(list) == 0 {
			fmt.Println("No bookmarks left.")
			return nil
		}
		id, ok, err := pickWithFzf(list, fields)
		if err != nil {
			return err
		}
		if !ok {
			return nil
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

// runSearchPrompt is the dependency-free fallback: query, list, act, repeat.
func runSearchPrompt(cfg Config, store *Store, fields SearchFields) error {
	label := fmt.Sprintf("Search %s (empty = all, 'q' to quit)", fields.Label())
	for {
		q := promptLine(label)
		if q == "q" {
			return nil
		}
		results := store.Search(cfg, q, fields, false)
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

// runPlainListLoop is runSearchPrompt's inner loop over a fixed list (deep search).
func runPlainListLoop(cfg Config, store *Store, list []*Bookmark) error {
	printResults(list)
	for {
		sel := promptLine("id to open/edit ('q' to quit)")
		if sel == "q" || sel == "" {
			return nil
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
		if actionMenu(cfg, store, b) == actionQuit {
			return nil
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
		opts += "  attachmen(t)s  (e)dit  (d)elete  (b)ack  (q)uit"
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
		case "t":
			attachmentsMenu(cfg, b)
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

// badgeSuffix renders " [md]", " [arc,att2]", " [att]", or "" -- see dev-docs.md#attachments.
func badgeSuffix(b *Bookmark) string {
	var parts []string
	if b.MarkdownFile != "" {
		parts = append(parts, "md")
	}
	if b.ArchiveFile != "" {
		parts = append(parts, "arc")
	}
	if n := len(b.Attachments); n == 1 {
		parts = append(parts, "att")
	} else if n > 1 {
		parts = append(parts, fmt.Sprintf("att%d", n))
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
