package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

// pickWithFzf renders the bookmark list to fzf: id hidden (field 1, used
// only for lookups), title/url/tags/folder visible and color-coded (fields
// 2-5) on the left, with a right-hand preview pane -- rendered by shelling
// back into `liber __preview <id>` -- showing the full title, url, tags,
// and description for whichever row is highlighted.
//
// It returns ok=false (no error) if the user cancelled (Esc/Ctrl-C/no
// match) rather than picking anything.
func pickWithFzf(list []*Bookmark) (id int, ok bool, err error) {
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
		fmt.Fprintf(&buf, "%d\t\x1b[1m%s\x1b[0m\t\x1b[34m%s\x1b[0m\t\x1b[33m%s\x1b[0m\t\x1b[32m%s\x1b[0m\n",
			b.ID, b.Title, b.URL, tags, folder)
	}

	previewCmd := fmt.Sprintf("%s __preview {1}", shellQuote(selfPath()))

	cmd := exec.Command("fzf",
		"--ansi",
		"--delimiter=\\t",
		"--with-nth=2,3,4,5",
		"--prompt=liber> ",
		"--header=title \u00b7 url \u00b7 tags \u00b7 folder    (preview: full details)",
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
	// --with-nth, so field 0 is always the hidden id.
	fields := strings.SplitN(line, "\t", 2)
	parsedID, convErr := strconv.Atoi(strings.TrimSpace(fields[0]))
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
