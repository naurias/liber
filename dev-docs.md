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

## Automation

`--auto` auto-classifies bookmarks by a case-insensitive substring match
against the URL (`ruleMatches` in `automation.go`), optionally setting a
folder and/or adding tags. The whole feature exists to satisfy one
constraint: **automation must never fight a later manual change.**

### The applied-rules ledger

Every bookmark carries `AppliedRules []AppliedAutoRule` — one entry per
rule that has ever been considered for it, recording the rule's id and
(if the rule set one) which folder it set. This is the mechanism that
makes everything else safe:

- **New bookmarks** (`resolveAutoRulesForNew`, called from both `create.go`
  and `import.go` before the bookmark is actually written): every
  currently active rule gets one look. A rule's folder only fills in an
  *empty* folder — so an explicit `-f` at creation (or a folder already
  present in a browser import) always wins over automation, matching the
  same "explicit beats convention" precedent as everywhere else in the
  tool. Tags are additive regardless.
- **Existing bookmarks** (`applyRulesToExisting`, used by `--auto add`'s
  automatic backfill and by `--auto apply`): only rules *not already in
  the ledger* are considered — that's what makes re-running a backfill
  (or adding a second, unrelated rule) safe to do repeatedly without ever
  reopening a decision that's already been made. Same empty-folder-only
  guard as above, which is exactly what makes a manual move stick: once a
  human (or a rule) has put something in a folder, no other rule will
  ever move it again through this path.

### Why editing a rule needed a second mechanism

The empty-folder guard above is deliberately conservative — once a
bookmark has *any* folder, it's hands-off. That's correct for "don't
reopen old decisions," but it means a plain reapply can't actually update
a bookmark to a rule's *new* folder value after an edit, since the
bookmark's folder is (by definition, if the rule ever applied) no longer
empty. This was caught directly during testing: editing a rule's target
folder and reapplying reported bookmarks as "reapplied" but silently left
their folder unchanged.

The fix is `reapplyRule`, used only by `--auto edit ... --reapply`. It
asks a more precise question than "is the folder empty": *does the
bookmark's current folder still exactly equal what this same rule set it
to last time?* That's exactly what the `Folder` field on each
`AppliedAutoRule` ledger entry is for. If yes — nothing has manually
touched it since — it's safe to advance the folder to the rule's new
value. If no — the folder differs from what this rule recorded, meaning
either the user moved it or a different rule claimed it first — reapply
leaves it alone, same as a plain apply would. Verified directly: reapplying
an edited rule across two bookmarks correctly updated the untouched one
and left a manually-moved one exactly where the user put it.

If a rule is edited such that it no longer matches a bookmark it
previously classified, `--reapply` drops that bookmark's ledger entry for
it entirely (via `removeAppliedRule`) — the bookmark's actual folder/tags
are left as they are; only the record of "this rule considered this
bookmark" is removed, since the rule no longer has anything to say about
it.

### Multiple folder rules, and deletion

If two folder rules could both match the same never-yet-classified
bookmark, whichever is processed first (rules are processed in ascending
id order) claims the folder; the second is still recorded in the ledger
(so it won't retry later) even though its folder action didn't take
effect. There's no priority/conflict system beyond creation order — this
is a deliberate scope cut for an edge case that's unlikely to matter in
practice.

`--auto delete` only removes the rule definition. It does not touch any
bookmark's current folder/tags or ledger entries — the classification a
rule already produced is treated as the bookmark's own state going
forward, same as if the user had set it by hand.

## Profiles

Every path liber writes to comes from `Config.effectiveBaseDir()`
(`config.go`), not `Config.BaseDir` directly:

```go
func (c Config) effectiveBaseDir() string {
    base := expandTilde(c.BaseDir)
    if c.ActiveProfile != "" {
        return filepath.Join(base, c.ActiveProfile)
    }
    return base
}
```

`htmlDir()`, `markdownDir()`, `archiveDir()`, and `indexPath()` all derive
from this (unless individually overridden by `html_dir`/`markdown_dir`/
`archive_dir` in config — those absolute overrides are deliberately NOT
profile-scoped, same "explicit wins" precedent as everywhere else). This
one function is the entire mechanism: every other file in the codebase
that computes a path already goes through these methods, so profile
isolation required no changes anywhere except the four places that used
to read `cfg.BaseDir` directly (`reindex.go`'s unindexed-quarantine and
staging roots, `sync.go`'s repo-root search) — those were switched to
`effectiveBaseDir()` too, so quarantine, staging, and sync are all
correctly profile-scoped as well.

`ActiveProfile == ""` means "no profile" — the original, pre-profiles flat
layout (`base_dir/html`, `base_dir/markdown`, etc. directly). This is
deliberate: existing users who never touch `--profile` see zero change in
behavior or file layout. `ActiveProfile`/`Profiles` live in `config.json`
(global settings), not in any profile's own `index.json` — switching
profiles never touches bookmark data, only which `index.json` subsequent
commands read.

Each profile's `index.json` is completely independent: its own id
sequence (`NextID`), its own automation rules (`NextAutoRuleID`/
`AutoRules`), tags, folders — everything. There is currently no mechanism
to move a bookmark between profiles or to search across more than one at
a time; "aware that others exist" is satisfied by `--profile` listing all
tracked profiles and marking the active one, not by any cross-profile
data operation.

`--profile <name>` creating a profile is genuinely just "add `name` to
`Config.Profiles` if not already there, set `ActiveProfile = name`" —
the actual directory is never created eagerly; it comes into existence
the first time something is written there, the same lazy
`os.MkdirAll`-on-write pattern every other directory in this codebase
already uses. `--profile delete` only removes the name from
`Config.Profiles` — it never touches the folder or its `index.json`, and
refuses to delete the currently active profile (to avoid `ActiveProfile`
silently pointing at an untracked name). Re-running `--profile <name>`
for a previously-deleted-from-tracking name picks its existing data back
up rather than starting fresh — `runProfileSwitch` checks the filesystem
(not just the tracked list) to decide whether to report "created" or
"switched", which was a real wording bug caught during testing: the first
version always said "Created" when a name wasn't in `Config.Profiles`,
even when its folder (and data) already existed on disk from before a
`--profile delete`.

## Deep search

`liber -s --deep` / `liber -sl --deep` add archive content as a second
match surface, on top of whatever field scope (`-sn`/`-sd`/etc, or the
default) is already active — see dev-docs.md#search-scoping for how that
scope is represented. A bookmark matches if its scoped metadata matches
**or** its archive content does; `--deep` never narrows what field
scoping already restricts, it only adds an extra way to match.

`extractArchiveText` (`deepsearch.go`) is a best-effort, dependency-free
HTML-to-text pass: strip `<script>`/`<style>` blocks first (their content
is markup/code, not page text), then strip every remaining tag, then
HTML-unescape what's left. This isn't a real text extractor — inline
`style="..."` attributes, SVG contents, and similar are simply removed
along with their enclosing tag, which is a feature here as much as a
limitation: single-file archives typically inline images as base64 data
URIs inside tag attributes (e.g. `<img src="data:...">`), and stripping
the whole tag conveniently discards that bulk along with the markup,
rather than searching in it. Reads are capped at `maxArchiveScanBytes`
(5MB) per file as a defensive limit against pathologically large archives
— a bookmark with a huge archive is only partially scanned past that
point.

**Why deep search needed its own control flow, not just a flag threaded
into the existing one:** the existing fzf-path relies on fzf itself doing
live, interactive fuzzy-filtering as the user types inside the picker —
there's no upfront "query" liber's side ever sees. Deep search is the
opposite: matching archive content requires a concrete literal string to
grep for *before* any list can be shown at all. The fix is
`promptDeepQuery`, called once by both `runSearch` and `runSearchLegacy`
when `deep` is set, producing a fixed candidate list that's then handed to
either `runSearchFzfList` (fzf over a static list, no live requery) or
`runPlainListLoop` (the plain picker over the same static list) — as
opposed to the non-deep path's `runSearchFzf`/`runSearchPrompt`, which
keep re-fetching `store.All()` or re-prompting on every loop iteration,
since there's no expensive upfront computation to avoid repeating there.

This split also fixed a real bug caught during testing: an earlier version
threaded a `deep` bool straight into the original `runSearchFzf`, which
prompted for the deep query *inside* the fzf-attempt function. When fzf
then failed at runtime (e.g. no controlling terminal) and control fell
through to the plain-prompt fallback, that fallback had no memory of the
already-typed query and asked for it again — the user had to type the
same (potentially expensive) query twice. Separating "compute the
candidate list once" from "which UI browses it" fixed this: the query is
now asked exactly once per invocation, before either UI is attempted.

## Web UI

`liber --serve` (`webui.go`) is `net/http` + `html/template` only — no
router, no JS framework, no external assets. Every page is rendered
server-side from Go string-constant templates parsed once at startup
(`layoutTmpl`/`searchBodyTmpl`), the same pattern `render.go` already used
for the html bookmark template. `html/template` (not `text/template`) is
what makes this safe against XSS from bookmark content the tool doesn't
control — a fetched page title or a description containing `<script>` gets
escaped automatically on render, same guarantee the CLI already relies on
for the per-bookmark html files.

**It deliberately reuses the CLI's own search machinery rather than
reimplementing it**: `handleSearch` calls the exact same `Store.Search`/
`filterDeep`/`SearchFields` that `-s`/`-sl`/`--deep` use, so web search
behavior can't drift from CLI behavior — a scope checkbox maps directly to
the same `n`/`u`/`t`/`d`/`f` letters as the CLI flags. `handleAdd` likewise
reuses `addBookmarkToStore` and `resolveAutoRulesForNew` directly, so a
web-added bookmark is indistinguishable from a CLI-added one: same
automation handling, same html/markdown/archive file generation.

**Duplicate detection needed a different UI than the CLI's y/n prompt**,
since a web request can't block on stdin. `handleAdd` renders the same
search page with the add form re-shown, pre-filled with what was
submitted, plus a hidden `confirm_dup=1` field and a relabeled submit
button ("Yes, add anyway") — resubmitting that exact form is the
confirmation. This mirrors `findDuplicate`'s CLI behavior (default: don't
add) without needing sessions, cookies, or JS. **Delete uses the identical
pattern**: `POST /delete` without `confirm=1` never deletes anything —
it renders a confirmation banner (via `renderDeleteConfirm`) with the
id/confirm fields already present as hidden inputs, and only the second,
explicit resubmission (with `confirm=1`) actually calls `deleteBookmarkFiles`/
`store.Delete`. This is also why delete is a `<form method="post">` per
row rather than a plain `<a href>` — a GET link can be prefetched or
crawled by the browser itself, which would be disastrous for something
destructive; a POST can't be triggered that way.

**Edit reuses the CLI's own edit machinery, not a parallel implementation**:
`handleEditSave` calls the exact same `syncBookmarkFiles` (folder moves +
file relocation), `addMarkdownCopy`, and `addArchiveCopy` that `liber -e`
uses — a web-edited bookmark's folder move physically relocates its files
identically to a CLI edit, and "add markdown/archive if missing" behaves
identically too (checkboxes only appear for whichever the bookmark doesn't
already have, same "only ever adds, never overwrites" rule as the CLI).
`GET /edit/{id}` and `POST /edit/{id}` share one handler
(`handleEdit`), branching on `r.Method` — the GET path only reads
(`store.Find` + render), so it needs no lock; the POST path
(`handleEditSave`) takes `writeMu` for its whole load-mutate-save sequence,
same as add and delete.

**Concurrency is a genuinely new concern here that the CLI never had**:
every CLI command is one process handling one request to completion, but
`net/http` runs each request in its own goroutine. Without synchronization,
two concurrent mutating requests could both load the same store state and
the second `Save()` would silently overwrite the first (a classic
lost-update race) — this applies equally to add, edit, and delete, so all
three take `writeMu sync.Mutex` around their entire load-mutate-save
sequence. Read-only handlers (`handleSearch`, the GET branch of
`handleEdit`, `handleArchive`, `handleMarkdown`) don't need it. This was
fixed proactively (identified during design, not caught as a bug
afterward) — verified with 5 sequential adds landing as ids 1-5 with no
gaps or collisions; true parallel-goroutine stress testing wasn't
achievable in the sandbox this was built in (background-process handling
was unreliable there), so the guarantee rests on the mutex covering the
whole critical section by construction, the standard correct fix for this
class of race.

**Profile-awareness required zero extra code**: every handler calls
`loadCfgAndStore()`, the same function every CLI command uses, which reads
`config.json` fresh per call — so the web UI automatically reflects
whichever profile is active, and even picks up a profile switch made via
the CLI in another terminal on its very next request, without needing a
server restart.

**Security default**: binds to `127.0.0.1:8080` unless `--addr` says
otherwise; binding to anything else prints a warning, since this server
has no authentication at all — anyone who can reach it can read, add,
edit, and delete anything in the collection.

## Web UI pagination

`paginate` (`webui.go`) is a pure function over an already-computed
`[]*Bookmark` — it doesn't know or care whether that list came from a
plain search, a scoped search, or `filterDeep`, which is what let
pagination apply uniformly to all three without any special-casing in
`handleSearch`. `webPageSize = 500` per what was asked: pagination is
invisible (no controls rendered at all) for any result set of 500 or
fewer, and only kicks in past that.

Page links (`pageURL`) are built by the handler, not the template —
`html/template` has no URL-construction helpers, so `handleSearch`
reconstructs `/` with the current `q`/`scope`/`deep` plus a new `page`
value using `net/url.Values`. This is what makes paging through a scoped
or deep search stay scoped/deep on every page, verified directly: a
`?q=page&scope=n` search's "next" link carries both params forward
unchanged.

**`renderSearchPageWithAddState` and `renderDeleteConfirm` (the views
shown when a duplicate-add or a delete needs confirming) always show page
1 of the *entire unfiltered* list**, not whatever page/search the user was
previously on. This is a deliberate simplification, not an oversight: it
matches behavior `handleAdd` already had before pagination existed (a
successful add already redirected to bare `/`, discarding any prior
search state), so extending the same simplification to the confirmation
views and to delete kept the three mutating paths consistent with each
other rather than making add special-cased. Precisely preserving "which
page of which search you were on" through a confirm-then-resubmit round
trip would need the current query state threaded through as hidden form
fields on every row's edit/delete controls; given these are transient,
one-off confirmation screens rather than primary navigation, that
precision wasn't worth the added form-field plumbing on every single
result row.

## Why not SQLite

Covered in README.md's "Why not SQLite?" section (user-facing rationale,
not implementation detail) — the short version is that id-renumbering
(the thing that might have motivated a real database) is fully solved in
plain Go by the staged-rename approach above, so there wasn't a technical
need to take on a CGO or large pure-Go SQL dependency for a single-user
local tool.
