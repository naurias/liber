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

// openURL opens a URL in the system's default browser.
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
