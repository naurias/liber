package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// use build flags or flake to override
var Version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "liber:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	if opt, legacy, ok := parseSearchFlag(args[0]); ok {
		if legacy {
			return runSearchLegacy(opt)
		}
		return runSearch(opt)
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage()
		return nil
	case "-v", "--version":
		fmt.Println("liber", Version)
		return nil
	case "-l", "--list":
		return runList()
	case "-e", "--edit":
		if len(args) < 2 {
			return fmt.Errorf("-e requires a bookmark id, e.g. liber -e 3")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid id %q: use `liber -l` to see valid ids", args[1])
		}
		return runEdit(id, args[2:])
	case "-d", "--delete":
		if len(args) < 2 {
			return fmt.Errorf("-d requires a bookmark id, e.g. liber -d 3")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid id %q: use `liber -l` to see valid ids", args[1])
		}
		return runDelete(id, args[2:])
	case "-r", "--reindex":
		return runReindex()
	case "__preview":
		// Internal: fzf's --preview callback 
		if len(args) < 2 {
			return nil
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return nil
		}
		return runPreview(id)
	case "config":
		return runConfigCmd(args[1:])
	}

	if strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("unknown flag %q (expected a URL first, or -s/-sl/-l/-e/-d/-r/-v/-h)", args[0])
	}

	// args[0] is the URL to bookmark.
	opt, err := parseCreateFlags(args[1:])
	if err != nil {
		return err
	}
	return runCreate(args[0], opt)
}

// liber search flags
func parseSearchFlag(flag string) (fields SearchFields, legacy bool, ok bool) {
	switch flag {
	case "--search":
		return SearchFields{}, false, true
	case "--search-legacy":
		return SearchFields{}, true, true
	}
	if !strings.HasPrefix(flag, "-s") {
		return SearchFields{}, false, false
	}
	for _, ch := range flag[2:] {
		switch ch {
		case 'l':
			legacy = true
		case 'n':
			fields.Title = true
		case 'u':
			fields.URL = true
		case 't':
			fields.Tags = true
		case 'd':
			fields.Description = true
		case 'f':
			fields.Folder = true
		default:
			return SearchFields{}, false, false // not a search flag at all
		}
	}
	return fields, legacy, true
}

func parseCreateFlags(args []string) (CreateOptions, error) {
	var opt CreateOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-i", "--interactive":
			opt.Interactive = true
		case "-md", "--markdown":
			opt.Markdown = true
		case "-a", "--archive":
			opt.Archive = true
		case "-t", "--tags":
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "-") {
				opt.Tags = append(opt.Tags, args[i])
				i++
			}
			i--
		case "-f", "--folder":
			if i+1 >= len(args) {
				return opt, fmt.Errorf("-f requires a folder name")
			}
			opt.Folder = args[i+1]
			i++
		default:
			return opt, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return opt, nil
}

func runConfigCmd(args []string) error {
	cfg, path, err := LoadConfig()
	if err != nil {
		return err
	}
	fmt.Println("Config file:", path)
	fmt.Println()
	fmt.Println("base_dir      ", cfg.BaseDir)
	fmt.Println("html_dir      ", cfg.htmlDir())
	fmt.Println("markdown_dir  ", cfg.markdownDir())
	fmt.Println("archive_dir   ", cfg.archiveDir())
	fmt.Println("singlefile_cmd", cfg.SingleFileCmd)
	if cfg.BrowserCmd != "" {
		fmt.Println("browser_cmd   ", cfg.BrowserCmd)
	}
	if cfg.EditorCmd != "" {
		fmt.Println("editor_cmd    ", cfg.EditorCmd)
	}
	fmt.Println("\nEdit the JSON file above to change these.")
	return nil
}

func printUsage() {
	fmt.Print(`liber - a small CLI bookmark manager

Usage:
  liber <url>                    save a bookmark
  liber <url> -i                 save interactively (prompts for description, tags, folder)
  liber <url> -md                also write a markdown copy
  liber <url> -a                 also write a full-page archive (requires the 'single-file' CLI)
  liber <url> -md -a             both markdown and archive
  liber <url> -t tag-a tag-b     attach tags at creation time
  liber <url> -f subfold         save into a subfolder of the base directory
  liber -s                       search/browse bookmarks, open or edit them
  liber -sn / -su / -st / -sd / -sf
                                  same, but restricted to one field: title / url / tags /
                                  description / folder (combine freely, e.g. -sdf = folder+description)
  liber -sl                      force the plain prompt (skip fzf even if installed)
  liber -sld                     legacy prompt restricted to descriptions (mix -l with any of n/u/t/d/f)
  liber -l                       list all bookmarks with their ids
  liber -e <id>                  edit a bookmark interactively
  liber -e <id> -t tag-a tag-b   set a bookmark's tags directly
  liber -e <id> -f subfold       move a bookmark to a different folder
  liber -d <id>                  delete a bookmark (asks for confirmation)
  liber -d <id> -y               delete without confirmation
  liber -r                       reindex: drop entries whose files were deleted
                                  outside liber (quarantining any surviving
                                  markdown/archive copy into <base_dir>/unindexed/),
                                  and renumber remaining ids to close gaps
  liber config                   show the active config file and its path
  liber -v                       print the version

Flags may be combined, e.g.:
  liber https://example.com -i -t news reading -f articles -md -a

liber -s uses fzf for picking if it's installed on PATH (title/url/tags/folder
on the left, a markdown/archive presence badge, and a full detail preview --
including description -- on the right), otherwise falls back to a plain
numbered prompt. Use -sl to force the plain prompt regardless of whether fzf
is installed. Either can be narrowed to specific fields with the n/u/t/d/f
letters shown above.

Config lives at $XDG_CONFIG_HOME/liber/config.json (created on first run).
`)
}
