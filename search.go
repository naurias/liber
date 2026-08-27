package main

import (
	"fmt"
	"strconv"
	"strings"
)

func runSearch() error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	if len(store.Bookmarks) == 0 {
		fmt.Println("No bookmarks yet. Add one with: liber <url>")
		return nil
	}

	if fzfAvailable() {
		if err := runSearchFzf(cfg, store); err != nil {
			fmt.Printf("(fzf picker failed: %v -- falling back to plain search)\n", err)
		} else {
			return nil
		}
	} else {
		fmt.Println("(tip: install fzf for a fuzzy picker here -- falling back to plain search)")
	}
	return runSearchPrompt(cfg, store)
}

// runSearchLegacy skips the fzf check entirely, for `liber -sl` -- useful
// if you have fzf installed but want the plain prompt anyway.
func runSearchLegacy() error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	if len(store.Bookmarks) == 0 {
		fmt.Println("No bookmarks yet. Add one with: liber <url>")
		return nil
	}
	return runSearchPrompt(cfg, store)
}

// runSearchFzf pipes the bookmark list to fzf for fuzzy selection, then
// hands the pick off to the same open/edit/delete action menu used by the
// plain-prompt path. It loops so edits/deletes are reflected next time the
// picker opens. A genuine fzf failure (as opposed to the user cancelling)
// is returned to the caller so it can fall back to the plain prompt.
func runSearchFzf(cfg Config, store *Store) error {
	for {
		all := store.All()
		if len(all) == 0 {
			fmt.Println("No bookmarks left.")
			return nil
		}
		id, ok, err := pickWithFzf(all)
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
func runSearchPrompt(cfg Config, store *Store) error {
	for {
		q := promptLine("Search title/url/tag/folder (empty = all, 'q' to quit)")
		if q == "q" {
			return nil
		}
		results := store.Search(q)
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
		choice := strings.ToLower(promptLine("(o)pen  (e)dit  (d)elete  (b)ack  (q)uit"))
		switch choice {
		case "o":
			if err := openURL(cfg, b.URL); err != nil {
				fmt.Println("Could not open browser:", err)
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
		fmt.Printf("[%d] %s\n     %s\n     folder:%s%s\n", b.ID, b.Title, b.URL, folder, tags)
	}
	fmt.Println()
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
