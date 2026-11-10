# gh-repopotamus

A `gh` extension that moves GitHub repos from one owner (user or org) to another in batches, and clones an owner's repos locally. It has a CLI and a two-pane TUI, and every feature works from both.

```bash
gh repopotamus move pborenstein documentation-rodeo --match 'site-*' --no-archived
```

It picks the repos, shows a plan, asks before changing anything, and keeps going if one repo fails.

## Install

There are no precompiled releases yet, so build from source (Go 1.27+):

```bash
git clone https://github.com/pborenstein/gh-repopotamus
cd gh-repopotamus
go build -o gh-repopotamus .
gh extension install .
```

Auth comes from `gh`: if `gh auth status` passes, so does repopotamus. Moving repos into an org requires permission to create repos there. `gh org list` shows the orgs you can use.

## Commands

| Command | Does |
|:--------|:-----|
| `gh repopotamus` | Opens the TUI (same as `tui`) |
| `gh repopotamus list OWNER [filters] [-l]` | Prints an owner's repo names, one per line |
| `gh repopotamus move SRC DST [REPO...] [flags]` | Transfers repos from SRC to DST |
| `gh repopotamus clone OWNER [REPO...] [flags]` | Clones repos into a local directory |

## Choosing repos

`move` and `clone` choose repos the same way, and `list` takes the same filters.

### Sources

| Source | How |
|:-------|:----|
| Names | Positional `REPO...` (`name` or `SRC/name`) |
| stdin | `-` as a positional, one name per line |
| File | `--from-file PATH`, one name per line (`#` comments and blank lines are ignored) |

Names from all sources are combined, then the filters narrow that set. With no names, the filters apply to every repo in the owner.

### Filters

| Filter | Matches |
|:-------|:--------|
| `--match GLOB` | Repo name, e.g. `'site-*'` |
| `--regex RE` | Repo name |
| `--topic T` | Has topic T (repeatable; all must match) |
| `--visibility V` | `public`, `private` or `internal` |
| `--language L` | Primary language, case-insensitive |
| `--pushed-before DATE` | Last push before `YYYY-MM-DD` |
| `--archived` / `--no-archived` | Only / exclude archived repos |
| `--forks` / `--no-forks` | Only / exclude forks |

## list

`-l` adds visibility, archived, fork, language, last push and topics to each line. Use `list` to check a selection before acting on it.

## move

| Flag | Does |
|:-----|:-----|
| `--dry-run` | Prints the plan and exits |
| `-y`, `--yes` | Skips the y/N prompt |
| `--all` | Required to select every repo in SRC (no names, no filters) |

A move runs in five steps:

1. Lists the repos in SRC and DST.
2. Builds a plan. A repo is skipped, with the reason shown, if its name isn't in SRC or DST already has a repo with that name.
3. Prints the plan and asks `y/N`. When the names came from stdin, the answer is read from the terminal.
4. Transfers one repo at a time. A failure is reported and the rest continue.
5. Prints moved, failed and skipped counts. Exits non-zero if any transfer failed.

GitHub completes a transfer asynchronously. Transferring the same repo again within about 10 seconds fails with 422, so wait before moving something back.

```bash
# Two named repos
gh repopotamus move pborenstein my-org site-blog site-docs

# Everything matching a glob, minus archived repos, as a dry run
gh repopotamus move pborenstein my-org --match 'site-*' --no-archived --dry-run

# Pipe a filtered list into move
gh repopotamus list pborenstein --pushed-before 2020-01-01 | gh repopotamus move pborenstein attic -

# Names from a file, no prompt
gh repopotamus move my-org pborenstein --from-file repos.txt --yes
```

## clone

`clone` runs `gh repo clone` into `DIR/REPO`, so gh's auth, protocol setting and fork upstreams apply. With no names and no filters, it clones every repo in OWNER. A repo is skipped if `DIR/REPO` already exists or the name isn't in OWNER.

| Flag | Does |
|:-----|:-----|
| `--dir DIR` | Clones into DIR (default: the current directory; `~` and relative paths work) |
| `--dry-run` | Prints the plan and exits |
| `-y`, `--yes` | Skips the y/N prompt |

```bash
# Every non-archived repo in an org
gh repopotamus clone my-org --dir ~/src/my-org --no-archived

# A hand-picked few
gh repopotamus clone pborenstein site-blog site-docs --dir ~/src
```

## TUI

Two panes, each showing one owner's repos. Queued moves change the panes right away, but nothing transfers until you confirm on the review screen.

| Key | Does |
|:----|:-----|
| `o` | Opens any owner in the active pane |
| `>` | Queues the selection (or the row under the cursor) to move to the other pane |
| `c` | Clones the selection, or every repo the pane shows if nothing is selected |
| `d` | Changes the clone directory on the clone review (`--clone-dir` sets it at launch) |
| `q` | Opens the review, grouped by source and destination |
| `y` | Runs the review |

The TUI's help line lists the rest, including type-to-filter, the archived and fork toggles, select all and invert.

## After moving: GitHub Pages

A moved repo's Pages site changes URL, from `OLD.github.io/repo` to `NEW.github.io/repo`, unless it uses a custom domain. The site needs a rebuild. If the generator bakes in its path (Eleventy `pathPrefix`, Jekyll `baseurl`), update that first.

Check how the site builds:

```bash
gh api repos/OWNER/REPO/pages -q .build_type
```

| `build_type` | Rebuild with |
|:-------------|:-------------|
| `workflow`, with a `workflow_dispatch` trigger | `gh workflow run WORKFLOW.yml -R OWNER/REPO` |
| `workflow`, no dispatch trigger | `gh run rerun RUN_ID -R OWNER/REPO` (find the ID with `gh run list -R OWNER/REPO -L 1`) |
| `legacy` (built from a branch) | `gh api -X POST repos/OWNER/REPO/pages/builds` |

## Development

| Task | Command |
|:-----|:--------|
| Offline tests | `go test ./...` |
| Build (gh picks up the new binary right away) | `go build -o gh-repopotamus .` |

Tests that really transfer a repo use the `integration` build tag and need a throwaway repo:

```bash
REPOPOTAMUS_TEST_REPO=pborenstein/test-repo REPOPOTAMUS_TEST_DEST=documentation-rodeo \
  go test -tags integration -v ./internal/github/
```

Project tracking lives in `docs/`: `CONTEXT.md` for current state, `IMPLEMENTATION.md` for the phase plan, `DECISIONS.md` for design decisions.

## Status

The CLI (`list`, `move`, `clone`) and the TUI work. Precompiled releases are next.
