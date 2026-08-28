package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runReindex reconciles the JSON index against the filesystem. It exists
// for the case where bookmark files were deleted (or moved) outside of
// liber -- e.g. `rm` from a file manager -- leaving index.json stale. It
// also closes any id gaps left behind by deletions, e.g. 1,2,4 -> 1,2,3.
//
// For each indexed bookmark:
//   - if its HTML file still exists, it's kept. Any Markdown/Archive path
//     it points to that no longer exists is simply cleared (nothing to move,
//     since it was already fully deleted).
//   - if its HTML file is gone, the bookmark is dropped from the index, and
//     any Markdown/Archive file it still points to is relocated -- not
//     deleted -- into <base_dir>/unindexed/{markdown,archive}/<same path>,
//     so an orphaned copy of an accidentally-deleted bookmark is never
//     silently destroyed.
//
// Every move above uses the exact relative path already recorded on that
// specific bookmark (set once at creation and kept in sync by edits) --
// never a filename-prefix or glob match across bookmarks. That's what
// guarantees a stray 0002 markdown file can never be paired with, or moved
// alongside, some other id's archive file: each record only ever touches
// its own recorded paths.
//
// Finally, surviving bookmarks are renumbered to close any gaps (their ids
// are reassigned sequentially in their current id order), and their
// html/markdown/archive files are renamed to match -- via a two-phase
// stage-then-commit move so a shifting id can never collide with a file
// that hasn't been moved out of the way yet.
func runReindex() error {
	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}

	unindexedRoot := filepath.Join(expandTilde(cfg.BaseDir), "unindexed")

	var kept []*Bookmark
	removed := 0
	orphansMoved := 0

	for _, b := range store.Bookmarks {
		htmlAbs := ""
		if b.HTMLFile != "" {
			htmlAbs = filepath.Join(cfg.htmlDir(), b.HTMLFile)
		}

		if htmlAbs != "" && fileExists(htmlAbs) {
			// Still a live, indexed bookmark. Just drop stale references
			// to md/archive copies that no longer exist -- there's
			// nothing to relocate, they're already gone.
			if b.MarkdownFile != "" && !fileExists(filepath.Join(cfg.markdownDir(), b.MarkdownFile)) {
				b.MarkdownFile = ""
			}
			if b.ArchiveFile != "" && !fileExists(filepath.Join(cfg.archiveDir(), b.ArchiveFile)) {
				b.ArchiveFile = ""
			}
			kept = append(kept, b)
			continue
		}

		// The bookmark's html was deleted outside liber: drop the index
		// entry, quarantining any surviving md/archive copy by its own
		// recorded path.
		removed++
		if b.MarkdownFile != "" {
			src := filepath.Join(cfg.markdownDir(), b.MarkdownFile)
			if fileExists(src) {
				dst := filepath.Join(unindexedRoot, "markdown", b.MarkdownFile)
				if err := moveFile(src, dst); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not move %s: %v\n", src, err)
				} else {
					orphansMoved++
				}
			}
		}
		if b.ArchiveFile != "" {
			src := filepath.Join(cfg.archiveDir(), b.ArchiveFile)
			if fileExists(src) {
				dst := filepath.Join(unindexedRoot, "archive", b.ArchiveFile)
				if err := moveFile(src, dst); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not move %s: %v\n", src, err)
				} else {
					orphansMoved++
				}
			}
		}
	}

	renamed, err := compactIDs(cfg, kept)
	if err != nil {
		return fmt.Errorf("renumbering ids: %w", err)
	}

	store.Bookmarks = kept
	store.NextID = len(kept) + 1
	if err := store.Save(); err != nil {
		return fmt.Errorf("saving index: %w", err)
	}

	fmt.Printf("Reindexed: %d bookmark(s) remain.\n", len(kept))
	if removed > 0 {
		fmt.Printf("Dropped %d entr%s whose bookmark file no longer exists.\n", removed, entrySuffix(removed))
	}
	if orphansMoved > 0 {
		fmt.Printf("Moved %d orphaned markdown/archive file(s) to %s\n", orphansMoved, unindexedRoot)
	}
	if len(renamed) > 0 {
		fmt.Println("Renumbered to close gaps:")
		for _, r := range renamed {
			fmt.Println("  " + r)
		}
	}
	if removed == 0 && orphansMoved == 0 && len(renamed) == 0 {
		fmt.Println("Nothing to clean up -- the index already matches what's on disk.")
	}
	return nil
}

func entrySuffix(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// bookmarkFileField abstracts over the three file kinds a bookmark can have
// so compactIDs can loop over them instead of repeating the same logic
// three times.
type bookmarkFileField struct {
	label string
	dir   func(Config) string
	get   func(*Bookmark) string
	set   func(*Bookmark, string)
}

var bookmarkFileFields = []bookmarkFileField{
	{"html", Config.htmlDir, func(b *Bookmark) string { return b.HTMLFile }, func(b *Bookmark, s string) { b.HTMLFile = s }},
	{"markdown", Config.markdownDir, func(b *Bookmark) string { return b.MarkdownFile }, func(b *Bookmark, s string) { b.MarkdownFile = s }},
	{"archive", Config.archiveDir, func(b *Bookmark) string { return b.ArchiveFile }, func(b *Bookmark, s string) { b.ArchiveFile = s }},
}

// compactIDs reassigns sequential ids (1..N) to kept, in their current id
// order, closing any gaps left by deletions. Renames happen in two phases:
// every affected file is first moved into a staging directory, and only
// once every bookmark's files have been staged are they moved to their
// final, renumbered names. Staging first means a bookmark being renumbered
// to id 3 can never collide with another bookmark that hasn't been moved
// off id 3 yet -- there's no ordering to get right, because nothing lands
// on its final name until every source file has already been moved aside.
func compactIDs(cfg Config, kept []*Bookmark) ([]string, error) {
	sort.Slice(kept, func(i, j int) bool { return kept[i].ID < kept[j].ID })

	type pendingMove struct {
		field     bookmarkFileField
		b         *Bookmark
		stagedAbs string
		finalRel  string
	}
	var pending []pendingMove
	var renamed []string

	stagingRoot := filepath.Join(expandTilde(cfg.BaseDir), ".liber", "restage")
	defer os.RemoveAll(stagingRoot)

	nextID := 1
	for _, b := range kept {
		oldID := b.ID
		newID := nextID
		nextID++
		if oldID == newID {
			continue
		}
		oldPrefix := fmt.Sprintf("%04d-", oldID)
		newPrefix := fmt.Sprintf("%04d-", newID)

		for _, f := range bookmarkFileFields {
			rel := f.get(b)
			if rel == "" {
				continue
			}
			base := filepath.Base(rel)
			if !strings.HasPrefix(base, oldPrefix) {
				fmt.Fprintf(os.Stderr, "warning: %s file for bookmark %d doesn't match the expected 0000- naming, leaving it as-is\n", f.label, oldID)
				continue
			}
			newBase := newPrefix + strings.TrimPrefix(base, oldPrefix)
			finalRel := filepath.Join(filepath.Dir(rel), newBase)

			srcAbs := filepath.Join(f.dir(cfg), rel)
			stagedAbs := filepath.Join(stagingRoot, f.label, rel)
			if err := moveFile(srcAbs, stagedAbs); err != nil {
				return renamed, fmt.Errorf("staging %s: %w", srcAbs, err)
			}
			pending = append(pending, pendingMove{field: f, b: b, stagedAbs: stagedAbs, finalRel: finalRel})
		}

		renamed = append(renamed, fmt.Sprintf("[%d] -> [%d]", oldID, newID))
		b.ID = newID
	}

	for _, p := range pending {
		finalAbs := filepath.Join(p.field.dir(cfg), p.finalRel)
		if err := moveFile(p.stagedAbs, finalAbs); err != nil {
			return renamed, fmt.Errorf("finalizing %s: %w", finalAbs, err)
		}
		p.field.set(p.b, p.finalRel)
	}

	return renamed, nil
}
