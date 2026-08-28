package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Fixed tab-delimited field positions fed to fzf. Field 1 (id) is never
// shown -- it exists purely so the preview callback and post-selection
// parsing can look a bookmark up exactly, without guessing from text.
const (
	fldID          = 1
	fldTitle       = 2
	fldURL         = 3
	fldTags        = 4
	fldFolder      = 5
	fldDescription = 6
	fldBadges      = 7
)

func fzfAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// selfPath returns the path to use for fzf's --preview callback: the
// currently running binary if resolvable, otherwise just "liber" (assumed
// to be on PATH, same as how the user is invoking us right now).
func selfPath() string {
	p, err := os.Executable()
	if err != nil {
		return "liber"
	}
	return p
}

// withNthFor computes fzf's --with-nth value for a given field scope.
// --with-nth controls both what's displayed AND what's fuzzy-matched, so
// restricting to (say) descriptions only narrows the search to that field
// and shows only that field. The markdown/archive badge (field 7) is
// always appended regardless of scope, since it's presence-at-a-glance
// info rather than something you'd search by.
func withNthFor(fields SearchFields) string {
	var idxs []string
	if !fields.Any() {
		idxs = []string{"2", "3", "4", "5"} // title, url, tags, folder -- unchanged default
	} else {
		if fields.Title {
			idxs = append(idxs, "2")
		}
		if fields.URL {
			idxs = append(idxs, "3")
		}
		if fields.Tags {
			idxs = append(idxs, "4")
		}
		if fields.Folder {
			idxs = append(idxs, "5")
		}
		if fields.Description {
			idxs = append(idxs, "6")
		}
	}
	idxs = append(idxs, "7") // badges always visible
	return strings.Join(idxs, ",")
}

// sanitizeField collapses embedded newlines/whitespace (which would break
// the one-record-per-line tab format) and truncates long text so a single
// bookmark can never make the picker unusable.
func sanitizeField(s string, maxLen int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		s = s[:maxLen] + "\u2026"
	}
	return s
}

// pickWithFzf renders the bookmark list to fzf, scoped to the requested
// fields (or the default title/url/tags/folder set if none are given),
// plus an always-visible markdown/archive presence badge. A right-hand
// preview pane -- rendered by shelling back into `liber __preview <id>` --
// shows the full title, url, tags, folder, and description for whichever
// row is highlighted.
//
// It returns ok=false (no error) if the user cancelled (Esc/Ctrl-C/no
// match) rather than picking anything.
func pickWithFzf(list []*Bookmark, fields SearchFields) (id int, ok bool, err error) {
	var buf bytes.Buffer
	for _, b := range list {
		folder := b.Folder
		if folder == "" {
			folder = "/"
		}
		tags := "(none)"
		if len(b.Tags) > 0 {
			tags = "#" + strings.Join(b.Tags, " #")
		}
		desc := sanitizeField(b.Description, 200)
		if desc == "" {
			desc = "(none)"
		}

		var badgeParts []string
		if b.MarkdownFile != "" {
			badgeParts = append(badgeParts, "md")
		}
		if b.ArchiveFile != "" {
			badgeParts = append(badgeParts, "arc")
		}
		badges := ""
		if len(badgeParts) > 0 {
			badges = "\x1b[2m[" + strings.Join(badgeParts, ",") + "]\x1b[0m"
		}

		fmt.Fprintf(&buf, "%d\t\x1b[1m%s\x1b[0m\t\x1b[34m%s\x1b[0m\t\x1b[33m%s\x1b[0m\t\x1b[32m%s\x1b[0m\t\x1b[36m%s\x1b[0m\t%s\n",
			b.ID,
			sanitizeField(b.Title, 200),
			sanitizeField(b.URL, 200),
			sanitizeField(tags, 200),
			sanitizeField(folder, 200),
			desc,
			badges,
		)
	}

	previewCmd := fmt.Sprintf("%s __preview {%d}", shellQuote(selfPath()), fldID)

	cmd := exec.Command("fzf",
		"--ansi",
		"--delimiter=\\t",
		"--with-nth="+withNthFor(fields),
		"--prompt=liber> ",
		"--header="+fields.Label()+"    (preview: full details)",
		"--height=80%",
		"--layout=reverse",
		"--border",
		"--no-multi",
		"--preview", previewCmd,
		"--preview-window=right:50%:wrap",
	)
	cmd.Stdin = &buf
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, runErr := cmd.Output()
	if runErr != nil {
		if exitErr, isExit := runErr.(*exec.ExitError); isExit {
			switch exitErr.ExitCode() {
			case 1, 130:
				// 1 = no match, 130 = interrupted (Esc/Ctrl-C).
				// Both mean "nothing picked" -- not a failure.
				return 0, false, nil
			default:
				// Anything else (fzf's own exit code 2 covers things like
				// no controlling terminal) is a genuine failure -- let the
				// caller fall back to the plain prompt instead of going
				// silent.
				msg := strings.TrimSpace(stderr.String())
				if msg == "" {
					msg = fmt.Sprintf("exit code %d", exitErr.ExitCode())
				}
				return 0, false, fmt.Errorf("%s", msg)
			}
		}
		return 0, false, runErr
	}

	line := strings.TrimRight(string(out), "\n")
	if line == "" {
		return 0, false, nil
	}
	// fzf's stdout is the original raw line (tab-delimited), regardless of
	// --with-nth, so the first field is always the hidden id.
	parts := strings.SplitN(line, "\t", 2)
	parsedID, convErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	if convErr != nil {
		return 0, false, nil
	}
	return parsedID, true, nil
}

// shellQuote wraps a path in single quotes for safe use inside the
// --preview command string fzf hands to /bin/sh -c, escaping any embedded
// single quote the POSIX way.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
