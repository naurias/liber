package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var htmlBookmarkTmpl = template.Must(template.New("bookmark").Funcs(template.FuncMap{
	"join": strings.Join,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<meta name="liber:id" content="{{.ID}}">
<meta name="liber:url" content="{{.URL}}">
<meta name="liber:folder" content="{{.Folder}}">
<meta name="liber:tags" content="{{join .Tags ", "}}">
<meta name="liber:created" content="{{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}">
<style>
  body { font-family: system-ui, -apple-system, sans-serif; max-width: 640px;
         margin: 3rem auto; padding: 0 1rem; color: #1a1a1a; background: #fafafa; }
  h1 { font-size: 1.4rem; line-height: 1.3; }
  a { color: #0b5fff; text-decoration: none; word-break: break-word; }
  a:hover { text-decoration: underline; }
  p.desc { color: #333; }
  .tag { display: inline-block; background: #eee; border-radius: 999px;
         padding: .15rem .6rem; margin: .2rem .3rem 0 0; font-size: .8rem; color: #444; }
  .meta { color: #888; font-size: .8rem; margin-top: 1.5rem; border-top: 1px solid #e2e2e2; padding-top: .75rem; }
</style>
</head>
<body>
  <h1><a href="{{.URL}}">{{.Title}}</a></h1>
  {{if .Description}}<p class="desc">{{.Description}}</p>{{end}}
  <div>{{range .Tags}}<span class="tag">#{{.}}</span>{{end}}</div>
  <p class="meta">Saved {{.CreatedAt.Format "Jan 2, 2006"}}{{if .Folder}} &middot; {{.Folder}}{{end}} &middot; id {{.ID}}</p>
</body>
</html>
`))

func writeHTMLBookmark(path string, b *Bookmark) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return htmlBookmarkTmpl.Execute(f, b)
}

func writeMarkdownBookmark(path string, b *Bookmark) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %q\n", b.Title)
	fmt.Fprintf(&sb, "url: %q\n", b.URL)
	fmt.Fprintf(&sb, "date: %s\n", b.CreatedAt.Format(time.RFC3339))
	if b.Folder != "" {
		fmt.Fprintf(&sb, "folder: %q\n", b.Folder)
	}
	if len(b.Tags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range b.Tags {
			fmt.Fprintf(&sb, "  - %s\n", t)
		}
	}
	sb.WriteString("---\n\n")
	fmt.Fprintf(&sb, "# %s\n\n", b.Title)
	if b.Description != "" {
		sb.WriteString(b.Description + "\n\n")
	}
	fmt.Fprintf(&sb, "[Visit original](%s)\n", b.URL)
	if b.ArchiveFile != "" {
		sb.WriteString("\n_A local archive of this page is also saved alongside it._\n")
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}
