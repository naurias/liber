package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// runSingleFile shells out to the `single-file` CLI to save a fully
// self-contained archive of a page. See:
// https://github.com/gildas-lormeau/single-file-cli
func runSingleFile(cmdName, url, outPath string) error {
	if cmdName == "" {
		cmdName = "single-file"
	}
	if _, err := exec.LookPath(cmdName); err != nil {
		return fmt.Errorf("%q not found in PATH — install it with `npm install -g single-file-cli`", cmdName)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	cmd := exec.Command(cmdName, url, outPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%s: %s", err, msg)
		}
		return err
	}
	return nil
}

var titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// fetchTitle makes a best-effort GET request and extracts <title>.
// Returns "" on any failure — callers should fall back to the URL itself.
func fetchTitle(rawURL string) string {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; liber-bookmark-manager/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 300*1024))
	if err != nil {
		return ""
	}
	m := titleRe.FindSubmatch(body)
	if m == nil {
		return ""
	}
	t := strings.TrimSpace(string(m[1]))
	t = html.UnescapeString(t)
	t = strings.Join(strings.Fields(t), " ")
	return t
}

// openURL opens a URL (or a local file path -- xdg-open/open handle both)
// in the system's default browser/handler. Fired and forgotten: suitable
// for GUI apps, not for anything that needs the current terminal.
func openURL(cfg Config, url string) error {
	if cfg.BrowserCmd != "" {
		return exec.Command(cfg.BrowserCmd, url).Start()
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// openInEditor opens a file in the user's preferred text editor: the
// config's editor_cmd if set, then $VISUAL, then $EDITOR, then finally
// falling back to the OS's default file association via openURL. Unlike
// openURL, an explicit editor is run synchronously with its stdio wired to
// the current terminal, since it may well be a terminal editor (vim, nano,
// emacs) that needs to take over the screen.
func openInEditor(cfg Config, path string) error {
	editor := cfg.EditorCmd
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor != "" {
		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	// No editor configured/set -- fall back to whatever the OS has
	// associated with this file type (may or may not be a text editor).
	return openURL(cfg, path)
}
