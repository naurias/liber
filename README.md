>[!Tip]
>This readme also serves as liber's documentation, if you simply want to check basic usage, [see here](#usage-overview)
# Introduction

Liber is a cross-platform, simple, private and local cli bookmark manager that saves bookmarks as browsable plain text HTML files, optionally  archives webpages and a markdown copy for bookmark's notes with configurable directories. It aslo has simple yet efficient webUI to work in browser. It uses simple json index (to not depend on database) to manage bookmarks behind the scenes.

<h2 style="text-align: center;">Available For</h2>
<div style="display: flex; gap: 20px;"> <div style="flex: 1;" align= center >Linux</div> <div style="flex: 1;" align= center >MacOS</div> <div style="flex: 1;" align= center>Windows</div> </div>

# Features 
- Plain text html bookmarks
- Tags and Directories, see [Design notes](#design-notes)
- Markdown copy of bookmarks for personal notes and additional description. 
	- You can simply store them as html and later create a markdown copy or archive later if you want to when editing a bookmark
- WebUI to browse add and edit bookmarks 
- Duplicate detection (tags, folder and bookmarks)
- Full archive of webpages (requires single-file-cli)
- Attcach files or related content with bookmarks 
- Check whether a bookmark has an archive and markdown copy or not 
- Powerful search and edit. Bulk management, deep search, missing archives. 
- Configurable location
- Import bookmarks from browser
- Rules based automation of bookmarks into specific folders or tags 
- Git integration for history and sync 
- Profiles, each working independantly from other

# Installing 
## Dependencies
### Install dependencies (Runtime)
Liber itself has no dependencies but some features require:
- [fzf](https://github.com/junegunn/fzf) (fuzzy finder), for live search
- [single-file-cli](https://github.com/gildas-lormeau/single-file-cli) for full web page archive
- git for history and syncing 
### Build dependencies 
To build liber from source you simply need 
- Go (version 1.22 or later)

## Archlinux 
- Install optional dependecies 
- Download the package build from [latest releases](https://github.com/naurias/liber/releases/latest/download/PKGBUILD) and install it, or simply:
```sh
wget https://github.com/naurias/liber/releases/latest/download/PKGBUILD
makepkgs -si
```

## NixOS 
This repo provides flake to install it on your nixos system. 
- To install it as flake add the repository to your flake in puts
  ```
  inputs = {
	  liber.url = "github:naurias/liber";
	  inputs.nixpkgs.follows = "nixpkgs";
  };
  ```
  and install it either on system with `environment.pkgs` or with home-manager `home.packages`
  ```nix
  {config, pkgs, inputs ... }: {
	  home.packages = [
		  inputs.liber.packages.${pkgs.stdenv.hostPlatform}.default
		  single-file-cli # optional
		  fzf # optional
	  ];
  }
  ```
  You can also test it on the go
  ```sh
  nix shell github:naurias/liber # enter a shell with liber
  # Or run binary directly 
  nix run github:naurias/liber
  ```
  
>[!Note]
>the nix and arch builds don't ship with optinal dependencies you'd have to install/declare them on your own
## Generic linux install 
- Download the binary from [latest releases](https://github.com/naurias/liber/releases/latest) or download directly from [here](https://github.com/naurias/liber/releases/latest/download/liber)
- Past the binary in your path
  ```sh
  /usr/local/bin
  # or 
  ~/.local/bin
  ```
## Build from Source 
Clone this repo and enter the directory:
```sh
git clone https://github.com/naurias/liber.git
cd liber
```
use either go to build or make file (not both)
To use go 
```sh
go build -o liber .
sudo mv liber /usr/bin/ #add it to the path
```
or use make file 
```sh
make && sudo make install
```
make sure you have optional dependencies if you want to use optional features 
## Windows 
- Download the `liber-setup.exe` from [latest releases] or use direct linke [here]
- Install as you'd do with any other exe file
- Open terminal and you can start using liber. If you want to use liber on terminal extensively on windows instead of using webUI `liber --serve` i'd recommend using windows terminal or [wezterm](https://wezterm.org/index.html)
## MacOS
- Install fzf and single-file-cli (either from theri respective github repos or homebres)
- clone the repository or download the source tar from latest release 
- Build using go and place the binary in your PATH
# Usage 
## Usage Overview
If you don't want to go through every usage detail here is the TLDR;
```
liber <url>                    save a bookmark
liber <url> -i                 save interactively (prompts for description, tags, folder)
liber <url> -md                also write a markdown copy
liber <url> -a                 also write a full-page archive (requires 'single-file')
liber <url> -md -a             both markdown and archive
liber <url> -t tag-a tag-b     attach tags at creation time
liber <url> -f subfold         save into a subfolder of the base directory
liber <url> -at report.pdf     attach a local file (repeatable; see "Attachments")
liber -s                       search/browse bookmarks, open or edit them (fzf if available)
liber -sn / -su / -st / -sd / -sf
                                same, but scoped to one field: title / url / tags / description / folder
liber -sdf                     fields combine freely, e.g. this is folder+description only
liber -sl                      force the plain prompt even if fzf is installed
liber -sld                     legacy prompt scoped to descriptions (mix -l with any of n/u/t/d/f)
liber -s --deep / -sl --deep   also full-text search inside archived pages (see "Deep search")
liber -l                       list all bookmarks with their ids
liber -e <id>                  edit a bookmark interactively (also offers to add a
                                markdown copy or archive if either is missing)
liber -e <id> -t tag-a tag-b   set a bookmark's tags directly (non-interactive)
liber -e <id> -f subfold       move a bookmark to a different folder (non-interactive)
liber -e <id> -md              add a markdown copy if it doesn't have one yet
liber -e <id> -a               add an archive if it doesn't have one yet
liber -e <id> -at report.pdf   attach a file to an existing bookmark (repeatable)
liber -e <id> -dt report.pdf   detach by name or number (deletes the saved copy)
liber -e <ids> ...             <id> can be a range/list too: 1-3, 2,5,3, or 1-4,7-9 --
                                applies the same flags (or interactive edit, one at a
                                time) to each matched bookmark; see "Batch operations"
liber -d <id>                  delete a bookmark (asks for confirmation)
liber -d <id> -y               delete without confirmation
liber -d <ids>                 <id> can be a range/list too, same as -e above
liber -r                       reindex: clean up + renumber (see "Reindexing" below)
liber --import <path>          import a browser bookmark export (see "Import" below)
liber --import <path> -md -a   same, also generating markdown/archives for each (slow)
liber --tags / --folders       list tags/folders with counts (see "Tag and folder hygiene")
liber --tags rename <a> <b>    rename a tag everywhere (merges if <b> already exists)
liber --tags delete <tag>      remove a tag from every bookmark that has it
liber --folders rename <a> <b> rename a folder (and its subfolders) everywhere
liber --folders delete <f>     move a folder's bookmarks back to the root
liber --history                list bookmarks by most recently opened (see "History")
liber --auto add --match <s> --folder <f> --tag <t1 t2>
                                auto-classify new bookmarks by URL (see "Automation")
liber --auto / --auto edit / --auto delete / --auto apply
                                list/edit/delete/re-run automations (see "Automation")
liber --profile                list profiles, with the active one marked (see "Profiles")
liber --profile <name>         switch to <name>, creating it if it's new
liber --profile default        switch back to the non-profile layout
liber --profile delete <name>  stop tracking a profile (its data is untouched)
liber --sync / --sync -p       commit (and optionally push) if it's a jj/git repo (see "Sync")
liber config                   show the active config file and its path
liber -v                       print the version
liber --serve                  local web UI for search + add (see "Web UI")
liber --serve --addr <host:port>
                                use a different address (default 127.0.0.1:8080)

```
## Detailed Usage 
This section provides detailed usage and examples

By default liber stores bookmarks in folder named Bookmarks created inside home folder. You can run empty command `liber` to list usage flags . 
- **Create a bookmark**
  ```sh
  liber <url>
  ```
- **Create a bookmark interactively**
  ```sh
  liber <url> -i
  ```
  This opens a prompt if you want to edit title name, add tags and put bookmark in specific subdirectory. There are individual command for quick tag or folder edits (listed below) but this command allows you to interact while bookmark creation 
- Create a bookmark along with **archive and or markdown copy**
  ```sh
  liber <url> -md # for markdown copy
  liber <url> -a # for archiving 
  ```
  This create a markdown copy and webpage archive for that url. Liber keeps archives and markdown copys in their respective folders of base bookmarks folder. The archive, markdown and bookmark are indexed and point to each other. For example a bookmark of google.com would have bookmark `indexnumber-google.com.html` in html folder while archive and markdown would live in archive and markdown folder respectively with same index. The index is dynamic and points to the bookmarks correctly even if one file is deleted the index would arrange them accordingly.
- Create bookmark with **specific tags or in specific sub directory **
  ```sh
  liber <url> -t tag-a tag-b # Use space to spearate tags 
  liber <url> -f folder-name # sub folder in which the bookmark goes 
  ```
- **Search** 
  ```sh
  liber -s 
  ```
  this flag allows you to search bookmarks interactively. If you have fzf and by default it would open a live preview otherwise a simple plain text prompt to allow you seach. By default the search would include everything i.e title, url, tags and folder. You can specify a search limited to tags, folders or title as . It also shows whether a bookmark has an archive or md copy or not 
  ```sh 
  liber -sn # title or name 
  liber -su # url 
  liber -st # to seach bookmarks with specific tags
  liber -sf # bookmarks in specific folder
  liber -sd # description
  # Seach flags can be combined as 
  liber -sfn # for folders and titles 
  ```
  It would fall back to plain test legacy search if you don't have fzf or you can force it by
  ```sh
  liber -sl
  # or you can combine the search flags 
  liber -sln # title or name 
  liber -slu # url 
  liber -slt # tags
  liber -sld # description
  liber -slf # folders
  # Search flags can be combined as 
  liber -sldf # for folders and descriptions
  ```
  Either way, picking a bookmark drops you into the open / edit / delete menu.
  Liber can also perform deep search to search for text inside of archived pages 
  ```sh
  liber -s --deep 
  ```
  `--deep` follows the criter of -s i.e any search flag lie `s, sn, sl and so on` can have --deep flag
- **List** all bookmarks with their id or index 
  ```sh
  liber -l
  ```
- **Edit** a bookmark (name, tag, folder, description or url).
  ```sh
  liber -e <id> 
  ```
  This allows you to edit a bookmark of given id. You can also edit bookmarks during search as selecting a bookmark during search `liber -s` can open a prompt whether to edit or not.  It also allows you to create a web archive or markdown of the specific bookmark if they aren't present. or you can edit something specific directly by:
  ```sh
  liber -e <id> -t tag-a tag-b # this will create tag-a and tag-b for that bookmark
  liber -e <id> -f folder-a # this will move bookmark to folder-a
  ```
- Add markdown or archive of bookmark that doesn't have 
  ```sh
  liber -e <id> -md # for markdown copy of that bookmark
  liber -e <id> -a # for archive 
  ```
  
  >[!Note]
  >The `<ids>` can be a range as well. for example `liber -e 1-3 -md` will create a markdown copy of bookmarks 1-3 it can also be comma separated list `liber -e 1-5,7,9-11` 
  >
- **Delete** a bookmark.
  ```sh
  liber -d <id> # delete bookmark
  ```
  the search flag `liber -s ` also opens up a prompt to delete bookmark as well.  **Just like editing the deletion id's can also be a range or comma separted list**
- **Attachments**
  ```sh
  liber <url> -at example-file # attach example-file to related bookmark
  liber -e <id> -at example-file # attach example-file to selected bookmark 
  liber -e <id> -dt example-file # remove attachment
  ```
  Liber allows attachments of files to the bookmarks with `-at` flag and their removal with `-dt` flag. Usefule for assossiating multiple archives related to bookmark or other related files 
- Show config and base directories 
  ```sh
  liber config 
  ```
  this would show where your config is and where bookmarks are stored 
- **Import** 
  ```sh
  liber --import <path>
  ```
  Liber can import bookmarks from browser's exported bookmarks file where `<path>` is the location of that file. See details in [Importing Bookmarks](#Importing-Bookmarks) section below
- **Tags and folder management** 
  ```sh
  liber --tags # list all tags and bookmarks in that tag (number)
  liber --folders # list all folders 
  liber --tags reaname <a> <b> # rename tag a to tag b, merge a to b if b is already present 
  liber --tags delete <tag> # delete tag
  liber --folders rename <a> <b> # same as tags 
  liber --folder delete <folder> # delete folder and move their bookmarks to root 
  ```
  There's no separate "merge" command, renaming _onto_ a name that already exists  **is** the merge: if a bookmark already has both the old and new tag, the rename just drops the old one rather than creating a duplicate. The same idea applies to folders (renaming `work` to `personal` when `personal` already has bookmarks just combines them). Folder rename/delete affects subfolders too (`work/urgent` follows `work` when you rename or delete it) and physically moves the affected files, the same as editing a single bookmark's folder does. Tag rename/delete rewrites the affected html/markdown files in place so their content stays consistent with the index.
- **History**
  ```sh
  liber --history # list bookmarks by most recently opened 
  ```
- **Automation/Rules**, see dedicated section [below](#automation)
  ```sh
  liber --auto add --match <s> --folder <f> --tag <t1 t2> 
  # auto classify new bookmarks by url <s>
  
  liber --auto edit # edit rules 
  liber --auto delete # delete rules 
  libet --auto apply # apply rules
  ```
- **Sync** 
  ```sh
  liber --sync # sync if bookmarks directory a git repo
  liber --sync -p # git push 
  ```
- **Profiles**: Liber also has profiles, By default there's just one collection, living directly under `base_dir`. If you want separate, fully independent collections say, `work` and `personal`, profiles give you that. see details [here](profiles).
``` sh
liber --profile                list profiles, with the active one marked (see "Profiles")
liber --profile <name>         switch to <name>, creating it if it's new
liber --profile default        switch back to the non-profile layout
liber --profile delete <name>  stop tracking a profile (its data is untouched)
```
- **WebUI**
  ```sh
  liber --serve 
  liber --serve --addr 127.0.0.1:8181 # use specific address for web server
  ```
  This opens a webui to manage bookmarks. Allows editing, searching and adding of bookmarks in browser window. By default it uses port 8080 of local host

>[!Note] Note
>The flags can be combined. For example `iber <url> -t tag-a -f folder-n` or `liber <url> -md -a` or `liber <url> -i -t news reading -f articles -md -a`
### Indexing and Reindexing 
Liber indexes bookmark id as simple json. They are id's that liber uses to identify and sync bookmarks, their markdown and archive copies. If you delete a bookmark the empty index would remain, adding further bookmarks would proceed without any problems but if you want to close the indices gap you can use:
```sh
liber -r 
```
to reindex the bookmarks list. It would set the index straight and sync up copies. It also checks for mismatch copies. For example if a bookmark is deleted and their markdown or archives aren't (directly in path by user outside of liber). liber would move missmatched copies to unindexed folder. This way you won't lose archives even if you delete anything in folders. Details explained below under configuration.

### Importing Bookmarks
`liber --import <path>` reads a browser bookmark export. Folders in the export become folders in liber (nested folders become `Parent/Child`); Firefox's per-bookmark `TAGS` and description are picked up too. Each imported bookmark gets a normal html file, exactly as if you'd run `liber <url>` and pass `-md`/`-a` to also generate markdown/archives for every import, though for a large export that's slow (archiving in particular makes one `single-file` call per bookmark) and probably better done selectively afterward with `liber -e <id> -md`/`-a`.

Anything that normalizes to a URL you already have is skipped automatically (no per-item prompt, unlike adding one bookmark at a time). So re-running `--import` on a refreshed export from your browser won't pile up duplicates. Entries with no `HREF` are skipped too. Both counts are reported at the end.

### Sync
`liber --sync` looks for a `.jj` or `.git` directory at or above `base_dir` and, if it finds one, commits the current state of your collection there (`liber --sync -p` also pushes afterward). It never initializes a repo itself. If there isn't one, it tells you and stops, since creating one unasked would be a strange thing for a bookmark tool to do. If `base_dir` is nested inside a larger repo (e.g. a dotfiles checkout), it still finds the right root.

This is deliberately minimal: one commit, optionally one push, nothing that manages branches/bookmarks(jj)/remotes for you. Since everything liber writes is flat files and JSON, git or jj sync was already going to work without this command, `--sync` just saves you the two-or-three manual commands. 
### Automation
Auto-classify bookmarks whose URL contains a given string, consider the exampl below:
```sh
liber --auto add --match doxy --folder hot
liber --auto add --match doxy --folder hot --tag important urgent
liber --auto                                  # list automations, with how many bookmarks each has classified
liber --auto edit <id> --folder other-folder  # change what a rule does
liber --auto edit <id> --folder x --reapply   # change it AND re-sync bookmarks it already classified
liber --auto delete <id>                      # remove a rule (doesn't undo what it already did)
liber --auto apply                            # re-run every rule against existing bookmarks
liber --auto apply <id>                       # re-run just one
```
everything from `doxy.com` should always land in a `hot` folder.
**Automation never overrides an explicit choice, and never re-opens a decision it’s already made.** Concretely:

- Creating a bookmark with `-f somefolder` (or importing one that already has a folder from your browser) always wins, a folder rule only ever fills in an _empty_ folder. Tags still get added either way, since tags are additive rather than exclusive.
- Adding a new rule immediately applies it to any existing bookmarks that match ( bookmarks created before the rule existed still get classified), but only bookmarks that don’t already have a folder. Anything already organized, by hand or by another rule is left alone.
- **Once a bookmark has been classified (automatically or manually) and you later move it yourself, that move sticks.** Automation tracks which rules have already had their one chance at each bookmark, so re-running `--auto apply` or adding an unrelated new rule never revisits a decision that’s already been made, including ones automation itself made earlier.
- Editing a rule updates its definition for future bookmarks; it does _not_ retroactively touch bookmarks it already classified unless you add `--reapply`. Even then, `--reapply` only advances a bookmark to the rule’s new value if the bookmark’s current folder still exactly matches what that same rule set it to last time, if you’ve moved it since, `--reapply` leaves it alone too.
- Deleting a rule removes the rule only; bookmarks it already classified keep their folder/tags exactly as they are.

### Web UI

`liber --serve` starts a local web UI at `http://127.0.0.1:8080` for searching, adding, editing, and deleting bookmarks.

>[!Tip]
>Add a bookmarklet to your browser’s toolbar with this as the URL (adjust the port if you used `--addr`) to quickly bookmark whatever page you’re currently on:
> ```
>javascript:location.href='http://127.0.0.1:8080/?prefill='+encodeURIComponent(location.href)
>```

It reflects whichever profile is active, and even picks up a profile switch made via the CLI in another terminal on its next request, no restart needed.

Once a search’s result count passes 500, simple `?page=N` pagination appears automatically (no controls at all below that). A scoped or deep search’s page-forward/back links carry the same query along, so paging through a filtered search keeps it filtered.
# Configuration 
By default or on first run, liber writes a default config to:
```sh
$XDG_CONFIG_HOME/liber/config.json    # usually ~/.config/liber/config.json
```
The configuration is simple and allows you to define your bookmarks directory and single file command (useful if you are using a different single command name or path)
```json
{
  "base_dir": "/home/you/Bookmarks",
  "singlefile_browser_path": "firefox",
  "singlefile_cmd": "single-file"
}
```

Fields:
- `base_dir`: root of your bookmark collection.
- `html_dir` / `markdown_dir` / `archive_dir` / `attachment_dir`: override any of the four subdirectories individually; each defaults to `<base_dir>/html`, `<base_dir>/markdown`, `<base_dir>/archive`, `<base_dir>/attachments`.
- `singlefile_cmd`: the executable used for `-a` archiving (default `single-file`).
- `singlefile_browser_path`:browser executable handed to single-file as --browser-executable-path on every archive run (leave unset to let single-file find the browser itself). Useful when single-file can't locate e.g. Brave: "singlefile_browser_path": "/usr/bin/brave".
- `browser_cmd`:  override the command used by `liber -s`'s "open"/"archive" actions (defaults to `xdg-open` / `open` / the Windows shell handler, by OS).
- `editor_cmd`:  override the command used by `liber -s`'s "markdown" action (defaults to `$VISUAL`, then `$EDITOR`, then the OS's default file association, in that order).

Run `liber config` to see the resolved paths.
>[!Warning]
>If archives are not being created make sure you have dependecies installed and browser path is set. The browser path can either be a command `firefox` or path to actual browser `/usr/bin/firefox`

## Layout Example 
```
<base_dir>/
  html/<folder>/0007-my-title.html
  markdown/<folder>/0007-my-title.md
  archive/<folder>/0007-my-title.html
  .liber/index.json
```

Every bookmark's files are prefixed with its numeric id, so `liber -l` / `liber -s` results always line up with what's on disk. Editing a bookmark's folder moves its files; editing anything else just rewrites them in place. Editing (interactively, or with `-md`/`-a`) can also _add_ a markdown copy or archive that wasn't there before — it reuses the bookmark's original id-slug basename (taken from its html file) so the new file lines up with the others exactly, even if the title has changed since creation. It only ever adds what's missing — an existing markdown copy or archive is left alone, not regenerated.

## Profiles
A profile is just a subfolder of `base_dir`, `liber --profile work` makes `<base_dir>/work/` the effective base dir for everything (`html/`, `markdown/`, `archive/`, `.liber/index.json`) until you switch again. Each profile has its own book1marks, ids, tags, folders, and [automations](#automations) , completely independent; there's currently no way to search or move bookmarks across profiles, only to list which ones exist and switch between them. `active_profile` is stored in `config.json`, which is otherwise shared (things like `editor_cmd`/ `browser_cmd` apply regardless of which profile is active).

Nothing changes if you never touch `--profile` — the original flat layout (`base_dir/html`, etc.) is exactly what "no active profile" means, so existing collections are completely unaffected.

`--profile delete` only stops tracking a profile in the list shown by `--profile`. It never touches the folder or its bookmarks, and refuses to delete the currently active profile (switch away first). Switching to a name you'd previously deleted-from-tracking picks its existing data back up rather than starting over.
## Reindexing

`liber -r` does two things, in order:

**1. Clean up entries whose files were deleted outside liber.** Every bookmark's Markdown/Archive paths are recorded individually on that bookmark's own index entry when it's created (and kept in sync whenever you edit it), liber never matches files across bookmarks by filename pattern. So:

- If you delete a bookmark's `.html` file yourself (`rm`, a file manager, etc.) instead of through `liber -d`/`liber -s`, the index still points at it and thinks it exists.
- `liber -r` checks every entry's recorded HTML path. If it's gone, that entry is dropped from the index, but its recorded Markdown/Archive files (if they're still there) are **moved, not deleted**, into `<base_dir>/unindexed/markdown/...` and `<base_dir>/unindexed/archive/...`, preserving their original relative path.
- Because each move follows that one bookmark's own recorded path rather than a glob/prefix match, a stray `0002-*.md` can never get relocated alongside, or confused with, some other id's `.html`/archive file, even if two bookmarks share a folder or similar-looking filenames.
- Bookmarks whose HTML is still present are left alone; only truly-missing Markdown/Archive references on those are cleared (there's nothing to move since they're already fully gone).

**2. Renumber to close id gaps.** If you had ids 1–4 and deleted 3, `liber -l` would otherwise show 1, 2, 4 forever. `liber -r` renumbers the remainder to 1, 2, 3, in their existing order, it's a gap-closing compaction, not an alphabetical or any other kind of sort. Since ids are embedded in filenames (`0004-...` → `0003-...`), this physically renames each affected bookmark's html/markdown/archive files to match. That rename is done in two passes, every affected file is moved to a temporary staging name first, and only once all of them are staged does anything land on its final numbered name, so a bookmark moving into a lower id slot can never collide with, or get confused with, another bookmark's files, no matter how many ids shift in the same run.

Safe to run any time. Step 1 never touches a bookmark whose HTML file is still there and never deletes anything outright, and step 2 only ever renames files, never their content.



# Design notes
## Why tags and folders both
The main reason is to make bookmarks organized even outside the use of liber 

Tags are useful for identification and classification of bookmarks pair them with folders it allows clean management. Folders can be used for per project or website based classification and tags to mark them for their relevance. It also prevent cluttering the bookmarks with too many tags and folder can provide a higher level domain for classifcation. For example a folder for a project can classify bookmarks cleanly while still having tags like study, important and so on. While true you can tag them on per project basis as well but liber is meant to be used as general bookmark manager that allows easy bookmark navigation even when you're exploring your bookmarks outside of liber

## Markdown Copy
Markdown copy is not meant to be whole html conversion or webpage contents of a url. It is simply meant to add personal notes, additional description, or your own personal bookmark specific content into it. For example you can explain what were you looking or what did you research in you markdown copy of that bookmark. For the purpose of archiving webpage single-file fills that role, so markdown has no need to to copy the contents of webpage  

## Why not a Database for Indexing 

The main reason is to keep it simple and plain go is fully able to fulfill this project's need. The indexing is fully solvable in plain Go, so there wasn't a technical need to bring in a database.  Liber is meant to be used as local simple cli bookmark manager. If project becomes too large for plain go to handle indices or isn't providing reasonable performance with it then it could be considered in future however, the goal is meant to keep it simple and exploitable even outside the use of Liber 

Sticking with flat JSON + files also keeps the project at zero external dependencies, keeps the whole collection readable/diffable/greppable and easy to back up or sync with git, and avoids either a CGO build dependency.
