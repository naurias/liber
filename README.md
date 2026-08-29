# liber

A small, dependency-free CLI bookmark manager. Saves bookmarks as browsable
HTML files (optionally with a Markdown copy and a full-page archive), keeps
a JSON index for fast search, and supports tags and folders.

Written in Go with **zero third-party dependencies** — just the standard
library. Compiles to a single static binary.

## Install

```sh
go build -o liber .
sudo mv liber /usr/local/bin/
```

Or with Nix (flake included):

```sh
nix build .
# or, to try it without installing:
nix run . -- https://example.com
```

Optional: for `-a` (archive), install [single-file-cli](https://github.com/gildas-lormeau/single-file-cli):

```sh
npm install -g single-file-cli
```

Optional: for a fuzzy picker in `liber -s`, install [fzf](https://github.com/junegunn/fzf).
Without it, `-s` falls back to a plain type-a-query prompt automatically.

## Usage

```
liber <url>                    save a bookmark
liber <url> -i                 save interactively (prompts for description, tags, folder)
liber <url> -md                also write a markdown copy
liber <url> -a                 also write a full-page archive (requires 'single-file')
liber <url> -md -a             both markdown and archive
liber <url> -t tag-a tag-b     attach tags at creation time
liber <url> -f subfold         save into a subfolder of the base directory
liber -s                       search/browse bookmarks, open or edit them (fzf if available)
liber -sn / -su / -st / -sd / -sf
                                same, but scoped to one field: title / url / tags / description / folder
liber -sdf                     fields combine freely, e.g. this is folder+description only
liber -sl                      force the plain prompt even if fzf is installed
liber -sld                     legacy prompt scoped to descriptions (mix -l with any of n/u/t/d/f)
liber -l                       list all bookmarks with their ids
liber -e <id>                  edit a bookmark interactively (also offers to add a
                                markdown copy or archive if either is missing)
liber -e <id> -t tag-a tag-b   set a bookmark's tags directly (non-interactive)
liber -e <id> -f subfold       move a bookmark to a different folder (non-interactive)
liber -e <id> -md              add a markdown copy if it doesn't have one yet
liber -e <id> -a               add an archive if it doesn't have one yet
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
liber --sync / --sync -p       commit (and optionally push) if it's a jj/git repo (see "Sync")
liber config                   show the active config file and its path
liber -v                       print the version
```

`liber -s` opens an [fzf](https://github.com/junegunn/fzf) picker if `fzf` is
on your `PATH`: the left side shows title/url/tags/folder plus a dim
`[md]`/`[arc]` badge for whichever of markdown/archive exist, all
color-coded and fuzzy-searchable, while the right side is a live preview
pane — rendered by liber itself — showing the full title, URL, tags,
folder, and description for whichever row is highlighted. The
badge is always shown regardless of search scope; it's presence-at-a-glance
info, not something you'd search by. Without fzf installed (or if it fails
to launch, e.g. no controlling terminal), it falls back to a plain
type-a-query / pick-an-id prompt (which also shows the badge); `liber -sl`
forces that fallback even when fzf is available. Either way, picking a
bookmark drops you into the same open / edit / delete menu.

Narrowing to specific fields (`-sn`/`-su`/`-st`/`-sd`/`-sf`, combinable, e.g.
`-sdf`) changes what's matched, not just what's shown — under fzf that's a
real search-scope change (`--with-nth`), not a cosmetic filter, so e.g.
`-sd` genuinely can't match on title text. The legacy prompt (`-sl` and its
combinations) restricts the same way via plain substring matching. `liber
-s` and `liber -sl` alone (no letters) keep matching everything, unchanged.

Flags can be combined:

```sh
liber https://example.com -i -t news reading -f articles -md -a
```

## Configuration

On first run, liber writes a default config to:

```
$XDG_CONFIG_HOME/liber/config.json    # usually ~/.config/liber/config.json
```

```json
{
  "base_dir": "/home/you/Bookmarks",
  "singlefile_cmd": "single-file"
}
```

Fields:

- `base_dir` — root of your bookmark collection.
- `html_dir` / `markdown_dir` / `archive_dir` — override any of the three
  subdirectories individually; each defaults to `<base_dir>/html`,
  `<base_dir>/markdown`, `<base_dir>/archive`.
- `singlefile_cmd` — the executable used for `-a` archiving (default
  `single-file`).
- `browser_cmd` — override the command used by `liber -s`'s "open"/"archive"
  actions (defaults to `xdg-open` / `open` / the Windows shell handler, by OS).
- `editor_cmd` — override the command used by `liber -s`'s "markdown" action
  (defaults to `$VISUAL`, then `$EDITOR`, then the OS's default file
  association, in that order).

Run `liber config` to see the resolved paths.

## Layout on disk

```
<base_dir>/
  html/<folder>/0007-my-title.html
  markdown/<folder>/0007-my-title.md
  archive/<folder>/0007-my-title.html
  .liber/index.json
```

Every bookmark's files are prefixed with its numeric id, so `liber -l` /
`liber -s` results always line up with what's on disk. Editing a bookmark's
folder moves its files; editing anything else just rewrites them in place.
Editing (interactively, or with `-md`/`-a`) can also *add* a markdown copy
or archive that wasn't there before — it reuses the bookmark's original
id-slug basename (taken from its html file) so the new file lines up with
the others exactly, even if the title has changed since creation. It only
ever adds what's missing — an existing markdown copy or archive is left
alone, not regenerated.

## Search

`liber -s` does a case-insensitive substring search across title, URL,
description, tags, and folder (or fuzzy, via fzf, if installed — see above),
then lets you act on the selected bookmark — looping until you quit. Scope
it to specific fields with `-sn`/`-su`/`-st`/`-sd`/`-sf` (combinable); force
the plain prompt with `-sl` (also combinable, e.g. `-sld` for
description-only via the plain prompt).

Picking a bookmark opens an action menu:

```
(o)pen  (m)arkdown  (a)rchive  (e)dit  (d)elete  (b)ack  (q)uit
```

`(m)` and `(a)` only appear when that bookmark actually has a markdown copy
or archive — trying them anyway (e.g. by habit) just says so rather than
erroring. `(o)` and `(a)` open in your browser (or `browser_cmd`, if set);
`(m)` opens in your editor (`editor_cmd`, else `$VISUAL`, else `$EDITOR`,
else the OS's default handler for `.md` files) and — unlike `(o)`/`(a)`,
which fire-and-forget — runs synchronously with the terminal handed to it,
since it might be a terminal editor like vim or nano that needs to take it
over.

## Batch operations

Both `-e` and `-d` accept a single id (as always), or a range/list:

```
liber -d 5              a single bookmark
liber -d 1-5            ids 1 through 5
liber -d 2,5,3          just those three (order doesn't matter)
liber -d 1-4,7-9        ranges and single ids can be mixed
liber -d 1-4, 7-9       spaces after commas are fine too
```

The same syntax works for `-e`, applying whatever flags you give to every
matched bookmark:

```
liber -e 1-3 -md        add a markdown copy to 1, 2, and 3 (skipping any that already have one)
liber -e 1-3 -a         same, for archives
liber -e 1-3 -f reading move all three into the "reading" folder
liber -e 1-3            no flags -- edits each one interactively, in turn
```

Deleting a range or list shows what's about to go and asks once for the
whole batch, rather than once per bookmark (`-y` skips that too, same as a
single delete). Editing a range or list with flags applies silently to
each one (no per-item prompts); editing one with no flags at all runs the
normal interactive prompts once per bookmark, with a header telling you
which one you're on. Either way, ids that don't exist are reported at the
end rather than aborting the rest of the batch.

## Reindexing

`liber -r` does two things, in order:

**1. Clean up entries whose files were deleted outside liber.** Every
bookmark's Markdown/Archive paths are recorded individually on that
bookmark's own index entry when it's created (and kept in sync whenever you
edit it) — liber never matches files across bookmarks by filename pattern.
So:

- If you delete a bookmark's `.html` file yourself (`rm`, a file manager,
  etc.) instead of through `liber -d`/`liber -s`, the index still points at
  it and thinks it exists.
- `liber -r` checks every entry's recorded HTML path. If it's gone, that
  entry is dropped from the index — but its recorded Markdown/Archive files
  (if they're still there) are **moved, not deleted**, into
  `<base_dir>/unindexed/markdown/...` and `<base_dir>/unindexed/archive/...`,
  preserving their original relative path.
- Because each move follows that one bookmark's own recorded path rather
  than a glob/prefix match, a stray `0002-*.md` can never get relocated
  alongside, or confused with, some other id's `.html`/archive file —
  even if two bookmarks share a folder or similar-looking filenames.
- Bookmarks whose HTML is still present are left alone; only truly-missing
  Markdown/Archive references on those are cleared (there's nothing to move
  since they're already fully gone).

**2. Renumber the survivors to close id gaps.** If you had ids 1–4 and
deleted 3, `liber -l` would otherwise show 1, 2, 4 forever. `liber -r`
renumbers the remainder to 1, 2, 3, in their existing order — it's a
gap-closing compaction, not an alphabetical or any other kind of sort.
Since ids are embedded in filenames (`0004-...` → `0003-...`), this
physically renames each affected bookmark's html/markdown/archive files to
match. That rename is done in two passes — every affected file is moved to
a temporary staging name first, and only once all of them are staged does
anything land on its final numbered name — so a bookmark moving into a
lower id slot can never collide with, or get confused with, another
bookmark's files, no matter how many ids shift in the same run.

Safe to run any time — step 1 never touches a bookmark whose HTML file is
still there and never deletes anything outright, and step 2 only ever
renames files, never their content.

### Why not SQLite?

The renumbering above is fully solvable in plain Go (that's what the
staged-rename does), so there wasn't a technical need to bring in a
database. Sticking with flat JSON + files also keeps the project at zero
external dependencies, keeps the whole collection readable/diffable/greppable
and easy to back up or sync with git, and avoids either a CGO build
dependency (`mattn/go-sqlite3`) or a much larger pure-Go driver
(`modernc.org/sqlite`) for a single-user local tool that doesn't need
concurrent multi-writer access or SQL querying. If you later want things
SQLite is genuinely good at — ad-hoc queries across a large collection,
multiple processes writing at once — that's a reasonable follow-up, just a
bigger one (new storage layer, migration from the existing `index.json`).

## Import

`liber --import <path>` reads a browser bookmark export — Firefox, Chrome,
and Safari all use the same Netscape Bookmark File Format (`Export
Bookmarks...` / `Export Bookmarks to HTML...`). Folders in the export
become folders in liber (nested folders become `Parent/Child`); Firefox's
per-bookmark `TAGS` and description are picked up too. Each imported
bookmark gets a normal html file, exactly as if you'd run `liber <url>` —
pass `-md`/`-a` to also generate markdown/archives for every import, though
for a large export that's slow (archiving in particular makes one
`single-file` call per bookmark) and probably better done selectively
afterward with `liber -e <id> -md`/`-a`.

Anything that normalizes to a URL you already have is skipped automatically
(no per-item prompt, unlike adding one bookmark at a time) — so re-running
`--import` on a refreshed export from your browser won't pile up
duplicates. Entries with no `HREF` are skipped too. Both counts are
reported at the end.

## Tag and folder hygiene

```
liber --tags                    list every tag with how many bookmarks use it
liber --tags rename <old> <new> rename a tag everywhere
liber --tags delete <tag>       remove a tag from every bookmark that has it

liber --folders                    list every folder with how many bookmarks are in it
liber --folders rename <old> <new> rename a folder (and its subfolders) everywhere
liber --folders delete <folder>    move that folder's bookmarks back to the root
```

There's no separate "merge" command — renaming *onto* a name that already
exists **is** the merge: if a bookmark already has both the old and new
tag, the rename just drops the old one rather than creating a duplicate.
The same idea applies to folders (renaming `work` to `personal` when
`personal` already has bookmarks just combines them).

Folder rename/delete affects subfolders too (`work/urgent` follows
`work` when you rename or delete it) and physically moves the affected
files, the same as editing a single bookmark's folder does. Tag
rename/delete rewrites the affected html/markdown files in place so their
content stays consistent with the index.

## History

Picking `(o)` in `liber -s`/`liber -sl` records when you opened a
bookmark's live URL (not its markdown/archive copy — that's a different
kind of interaction). `liber --history` lists everything you've opened,
most recent first, with an open count. Nothing is recorded until the first
time you use `(o)`.

## Sync

`liber --sync` looks for a `.jj` or `.git` directory at or above
`base_dir` and, if it finds one, commits the current state of your
collection there (`liber --sync -p` also pushes afterward). It never
initializes a repo itself — if there isn't one, it tells you and stops,
since creating one unasked would be a strange thing for a bookmark tool to
do. If `base_dir` is nested inside a larger repo (e.g. a dotfiles
checkout), it still finds the right root.

This is deliberately minimal: one commit, optionally one push, nothing
that manages branches/bookmarks(jj)/remotes for you. Since everything liber
writes is flat files and JSON, git or jj sync was already going to work
without this command — `--sync` just saves you the two-or-three manual
commands. Note for jj users: jj's own concept also called "bookmarks" (its
branch-like refs) is unrelated to liber's bookmarks — just a naming
coincidence to be aware of if you ever script the two together.

## Duplicate detection

Adding a URL that normalizes to one you already have (case-insensitive
scheme/host, default ports and a trailing slash stripped, common tracking
parameters like `utm_*`/`fbclid`/`gclid` removed) shows you the existing
entry and asks before adding a second one — defaulting to no. Path and
query *values* are never touched, only stripped/lowercased where it's safe
to, so this only ever catches URLs that are genuinely the same page, never
ones that just look similar. `liber --import` uses the same check but skips
silently instead of prompting, since bulk-importing isn't a good time to
ask per-item.

## Design notes / limitations

- Title fetching is a best-effort plain HTTP GET + `<title>` regex extraction
  (no JS rendering) — it falls back to the raw URL if the page can't be
  fetched or has no title.
- Markdown export is metadata + a link, not a full content-to-Markdown
  conversion (that generally needs a JS-capable renderer, which is what `-a`
  covers via `single-file`).
- The `-a` archive requires `single-file` to be installed separately;
  without it, liber logs a warning and still saves the HTML/Markdown parts.
- `liber __preview <id>` is an internal command (fzf's `--preview` callback
  for `liber -s`) — not meant to be run by hand, though it's harmless if you
  do; it just prints one bookmark's details.
- `liber -v` prints `dev` unless built with `-ldflags "-X main.Version=x.y.z"`
  (the Makefile and flake both set this for you).
- Minor asymmetry: the unrestricted `liber -s` (fzf) scope is title/url/tags/folder,
  while unrestricted `liber -sl` (plain prompt) also matches description. This
  predates field-scoping and is a fzf mechanics limitation (`--with-nth`
  ties display and match-scope together, so including description by
  default would force it into the visible columns too) rather than a
  deliberate difference — use `-sd`/`-sld` to search descriptions explicitly
  either way.
- `liber --sync`'s git path is well-tested (init, first commit, no-op
  "nothing to commit", push failure, and a nested-repo base_dir all behave
  correctly). Its jj path is implemented against jj's documented commands
  (`jj commit`, `jj git push`) but hasn't been run against a real jj repo —
  sanity-check it before relying on it.
- Import assumes a well-formed Netscape Bookmark File Format export (what
  Firefox/Chrome/Safari actually produce) and parses it with a handful of
  targeted regexes rather than a full HTML parser, to stay dependency-free —
  a hand-edited or otherwise unusual export might not parse cleanly.
