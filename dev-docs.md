# liber dev docs

Design notes, gotchas, and references for maintaining liber. Code comments
point here (`see dev-docs.md#topic`) instead of carrying the full
explanation inline — this file is where that explanation actually lives.

User-facing docs (install, usage, config) are in README.md. This file is
for whoever is reading or changing the source.

## Data model

Every bookmark's `HTMLFile` (always present), `MarkdownFile`, and
`ArchiveFile` are relative paths, stored individually on that bookmark's
own record. There is no filename-pattern or glob matching anywhere in the
codebase to relate one bookmark's files to another's — every operation
that touches a bookmark's files follows its own recorded path. That single
invariant is what makes several other features safe (see Reindex below):
a stray `0002-*.md` can never get paired with, or moved alongside, some
other id's `.html`/archive file, because nothing ever goes looking for
files by pattern.

Filenames follow `%04d-<slug>` (id, then a slugified title, e.g.
`0007-my-title.html`). The basename is fixed at creation time
(`create.go`'s `addBookmarkToStore`) and is **never re-slugged** by an
edit — editing only ever moves the *directory* (on a folder change), never
renames the file to match a new title. `edit.go`'s `sharedBase` depends on
this: when adding a markdown/archive copy to a bookmark that didn't have
one, it derives the shared basename from the existing HTML file rather
than re-deriving a slug from the (possibly since-edited) title, so all
three file kinds for one bookmark always share the same basename.

## Reindex

`liber -r` runs two independent passes, in order:

**1. Orphan cleanup.** For each indexed bookmark, check its recorded HTML
path. If it's gone (deleted outside liber, e.g. via `rm` or a file
manager), drop the index entry — but if its recorded Markdown/Archive file
is still there, move it (don't delete it) into
`<base_dir>/unindexed/{markdown,archive}/<same relative path>`. This uses
the exact path already recorded on that specific bookmark, never a
prefix/glob match — see Data model above for why that matters here.
Bookmarks whose HTML is still present are left alone, except that a
Markdown/Archive reference pointing at a file that's *also* gone gets
cleared (nothing to move, it's already gone).

**2. Id compaction.** Surviving bookmarks are renumbered to close any gaps
left by deletions (e.g. ids 1,2,4 → 1,2,3), in their existing order. Since
ids are embedded in filenames, this means physically renaming files.

The renaming is done in two phases — implemented in `reindex.go`'s
`compactIDs`:

1. **Stage**: every affected file (across all bookmarks being renumbered)
   is moved to a temporary staging name first.
2. **Commit**: only once *everything* has been staged does anything get
   moved to its final, renumbered name.

Why two phases instead of just renaming in id order: staging first means
there's no ordering to reason about. A bookmark moving to id 3 can't
collide with another bookmark's file that hasn't been moved off id 3 yet,
because nothing lands on a final name until every source file involved in
the whole batch has already been moved out of the way. (An ascending-order
single-pass rename would actually also be provably safe here — new ids are
always ≤ old ids, so processing in ascending order never creates a
collision — but the staged version doesn't require that argument to hold
for correctness, which makes it more robust against future changes to the
processing order.)

## Duplicate detection

`dedupe.go`'s `normalizeForDedupe` builds a comparison key by:

- lowercasing scheme and host (case-insensitive per the URL spec — this is
  the *only* case-folding done; path and remaining query **values** are
  left exactly as given, since those can be genuinely case-sensitive on
  the server)
- stripping the default port for the scheme (`:80` for http, `:443` for
  https)
- stripping a trailing slash from the path
- stripping known tracking query parameters (`utm_*`, `fbclid`, `gclid`,
  `mc_cid`, `mc_eid`, `igshid`, `ref`) and dropping the fragment entirely

This only ever collapses URLs that are genuinely the same resource, never
ones that merely look similar — the deliberate asymmetry (case-fold
scheme/host but never path/query) is what keeps false positives out.

`liber <url>` warns and asks before adding a detected duplicate (default:
no). `liber --import` skips duplicates silently instead, since prompting
per-item during a bulk import of potentially hundreds of bookmarks isn't
practical.

## Import format

The Netscape Bookmark File Format (what Firefox/Chrome/Safari all export
to) isn't real HTML — browsers emit it as a fixed, predictable sequence of
a handful of tags. `import.go`'s `netscapeTagRe` scans for exactly these,
in the order they appear:

```
<DT><H3 ...>Folder Name</H3>   -- names the NEXT <DL><p> that opens
<DL><p>                        -- enters that folder (or root, if no <H3> preceded it)
<DT><A HREF="..." TAGS="a,b">Title</A>
<DD>Optional description line  -- Firefox-only, follows its <A>
</DL>                          -- leaves the current folder
```

A small regex-driven scan is enough for this fixed format; pulling in a
full HTML parser would be overkill and — being an external dependency —
against the project's zero-dependency design. `parseNetscapeBookmarks`
tracks the `<H3>`/`<DL>` nesting with a simple stack to compute each
link's folder path (joined with `/`, e.g. `Work/Reference`).

## Tag and folder hygiene

There's no separate "merge" command for `--tags`/`--folders`. Renaming
*onto* a name that already exists **is** how merging is expressed: if a
bookmark already has both the old and new tag, the rename just drops the
old one (via `dedupe`) rather than creating a duplicate. Same idea for
folders — renaming `work` into an already-populated `personal` just
combines them, since a folder is nothing but a shared string on each
bookmark, not a distinct object with its own identity to merge.

Folder rename/delete affects subfolders too (`folderMatchesOrIsChild` /
`renameFolderPrefix` in `taxonomy.go` handle the prefix matching) and
physically moves files via the same `syncBookmarkFiles` helper `edit.go`
uses for a single bookmark's folder change. Tag rename/delete calls
`syncBookmarkFiles(cfg, b, false)` (folder unchanged) purely to get the
html/markdown content rewritten in place to reflect the new tag text.

`--folders delete <f>` is implemented as `runFoldersRename(f, "")` —
renaming to the root. There's no dedicated delete code path.

## History

`LastOpenedAt`/`OpenCount` on `Bookmark` are updated only by the `(o)`
action in the search picker's action menu (`search.go`) — i.e. actually
visiting the live URL. Opening a saved markdown copy `(m)` or archive
`(a)` is deliberately not tracked as a "visit"; it's a different kind of
interaction (reviewing your own saved copy, not browsing).

## Sync

`liber --sync` walks upward from `base_dir` (`sync.go`'s `findRepoRoot`,
up to 40 levels) looking for a `.jj` or `.git` directory, so it works
whether `base_dir` itself is the repo root or just a subdirectory of a
larger repo (e.g. a dotfiles checkout). It never initializes a repo
itself — if it doesn't find one, it says so and stops, since silently
creating a VCS repo would be a surprising thing for a bookmark tool to do.

Scope is deliberately minimal: one commit, optionally one push (`-p`),
nothing that manages branches, jj bookmarks, or remotes. Since everything
liber writes is flat files and JSON, git/jj sync was already going to work
without any special support from liber — `--sync` just saves the
two-or-three manual commands.

**Testing status**: the git path is well-exercised (init, first commit,
git's "nothing to commit" no-op detection, push failure surfacing git's
real error, and a nested-repo `base_dir` all verified). The jj path
(`jj commit`, `jj git push`) is implemented against jj's documented CLI
but has not been run against a real jj repo — jj wasn't available in the
environment this was built in. Sanity-check it before relying on it.

Note: jj's own concept also called "bookmarks" (its branch-like refs) is
entirely unrelated to liber's bookmarks — a naming coincidence worth
knowing about if the two are ever scripted together.

## fzf integration

`liber -s` uses fzf when it's available. This section covers the mechanics.

### Field layout

`fzf.go` feeds fzf tab-delimited lines with a fixed 7-field layout:

```
1: id (always hidden -- lookup key only)
2: title      3: url         4: tags
5: folder     6: description 7: badges (always visible)
```

Field 1 (id) is never displayed; it exists purely so the preview callback
and post-selection parsing can identify a bookmark exactly, without
guessing from displayed text.

### `--with-nth` ties display and match scope together

fzf's `--with-nth` controls **both** what's displayed **and** what's
fuzzy-matched — restricting to a field also restricts search to it, which
is exactly what field-scoped search (`-sn`/`-sd`/`-st`/`-sf`/`-su`) relies
on (`withNthFor` in `fzf.go` computes the field-index list per scope).

The flip side: this is also why the **default**, unrestricted `-s` scope
(title/url/tags/folder, `--with-nth=2,3,4,5`) doesn't include description,
even though the plain-prompt fallback's default *does* search description.
Including description in the default fzf scope would force it into the
visible columns too (there's no way with `--with-nth` alone to match on a
field without displaying it) — so the two search paths have a small,
known asymmetry here. Field 7 (badges) is always appended to whatever
`--with-nth` value is computed, since it's presence-at-a-glance info, not
something you'd want to search by.

### The `--delimiter` gotcha

**`--delimiter` needs the regex escape `\t` (backslash, t — two
characters), not a literal tab byte**, when combined with `--nth`/
`--with-nth`. Passing a literal tab silently breaks all matching (fzf just
returns no results, no error). This was confirmed directly against a real
fzf binary — a literal-tab delimiter matched nothing, the same command
with `--delimiter='\t'` worked immediately. `fzf.go` builds the flag as
`"--delimiter=\\t"` (a literal `\` followed by `t` in the resulting
argument string) for exactly this reason.

### Preview callback

The right-hand preview pane is rendered by fzf shelling back into the
liber binary itself: `--preview '<path-to-liber> __preview {1}'`, where
`{1}` is fzf's placeholder for the raw (never display-transformed) first
field — the hidden id. `main.go` routes the internal `__preview <id>`
subcommand to `preview.go`'s `runPreview`, which re-reads the index and
prints a labeled Title/URL/Tags/Folder/Description block. `selfPath()`
uses `os.Executable()` to get an exact path to the running binary rather
than assuming `liber` is on `PATH`.

This pattern (shell back into your own binary for a preview) is how most
non-trivial fzf integrations handle rendering something fzf itself can't
compute — e.g. git log pickers doing `--preview 'git show {1}'`.

### fzf's stdout is always the raw line

Regardless of `--with-nth`, fzf's stdout on selection is the full original
input line, not the display-transformed version. `pickWithFzf` relies on
this: it always splits the returned line on tab and reads field 0 (the
id), even when `--with-nth` was hiding most of the other fields from view.

### Exit code handling

fzf exit codes: `0` = picked something, `1` = no match, `130` = interrupted
(Esc/Ctrl-C). `1` and `130` are both treated as "user cancelled, not an
error." Anything else (fzf's own `2`, which covers things like no
controlling terminal) is treated as a genuine failure and returned to the
caller, which falls back to the plain prompt with a visible message rather
than silently doing nothing — this was a real bug caught during testing
(a no-TTY environment produced exit 2, which an earlier version of this
code was lumping in with "cancelled," making `-s` silently do nothing).

## Search scoping

`main.go`'s `parseSearchFlag` recognizes `-s`, `-sl`, and combinations:

```
-s               search everything (fzf if available)
-sl              same, but force the plain prompt
-s[nutdf]+       restrict to specific field(s): n=title, u=url, t=tags, d=description, f=folder
-sl[nutdf]+      same restriction, forced to the plain prompt
```

Letters combine freely and in any order (`-sdf` and `-sfd` are
equivalent). `SearchFields` (in `store.go`) is the shared representation
threaded through both the plain-prompt path (`Store.Search`, plain
substring matching per selected field) and the fzf path (`withNthFor`,
see above).

## Batch operations

`-e`/`-d` both accept a single id or a spec (`idspec.go`): a comma-separated
list of ids and/or `lo-hi` ranges, e.g. `1-5`, `2,5,3`, `1-4,7-9`. Reversed
ranges (`5-2`) are swapped rather than rejected; duplicates from
overlapping ranges/repeats are deduped; the result is sorted ascending for
predictable display order (input order doesn't otherwise matter — deleting
or editing one id has no effect on any other).

**Shell-splitting gotcha**: `liber -d 1-4, 7-9` (unquoted, with a space
after the comma) gets split by the shell into two argv entries, `"1-4,"`
and `"7-9"`, *before* liber ever sees them. `consumeIDSpec` handles this by
concatenating consecutive non-flag arguments (stopping at the first token
starting with `-`) rather than only ever looking at a single argv slot —
so the comma that's already attached to the first token joins cleanly with
the second, reconstructing `"1-4,7-9"`. This was a deliberate defensive
choice rather than requiring the user to quote the spec.

`-d` shows one aggregate confirmation for a multi-id batch (listing every
title about to be deleted) rather than confirming per-item; a single id
keeps the original single-line confirmation unchanged. `-e` with flags
(`-t`/`-f`/`-md`/`-a`) applies them to every matched bookmark silently, no
per-item prompts; `-e` with no flags at all runs the normal interactive
edit once per matched bookmark in turn, printing a `n of N` header between
them. Ids that don't exist in the store are collected and reported once at
the end rather than aborting the rest of the batch.

## Why not SQLite

Covered in README.md's "Why not SQLite?" section (user-facing rationale,
not implementation detail) — the short version is that id-renumbering
(the thing that might have motivated a real database) is fully solved in
plain Go by the staged-rename approach above, so there wasn't a technical
need to take on a CGO or large pure-Go SQL dependency for a single-user
local tool.
