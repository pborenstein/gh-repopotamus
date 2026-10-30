# gh-repopotamus

A `gh` extension that moves GitHub repos from one owner (user or org) to another, in batches.

```
gh repopotamus move pborenstein documentation-rodeo --match 'site-*' --no-archived
```

It picks the repos, shows a plan, asks before transferring anything, and keeps going if one transfer fails.

## Install

There are no precompiled releases yet, so build from source (Go 1.27+):

```
git clone https://github.com/pborenstein/gh-repopotamus
cd gh-repopotamus
go build -o gh-repopotamus .
gh extension install .
```

Auth comes from `gh`. If `gh auth status` is happy, so is repopotamus. To move repos into an org, you need permission to create repos there.

## Usage

### list

```
gh repopotamus list OWNER [filters] [-l]
```

Prints the owner's repos, one name per line. `-l` adds visibility, archived, fork, language, last push and topics. Takes the same filters as `move`, so you can check a selection before acting on it.

### move

```
gh repopotamus move SRC DST [REPO...] [flags]
```

Choosing repos:

| Source | How |
|--------|-----|
| Names | positional `REPO...` (`name` or `SRC/name`) |
| stdin | `-` as a positional, one name per line |
| File | `--from-file PATH`, one name per line (`#` comments and blank lines are ignored) |

Names from all sources are combined. The filters then narrow that set. With no names, the filters apply to every repo in SRC.

| Filter | Matches |
|--------|---------|
| `--match GLOB` | repo name, e.g. `'site-*'` |
| `--regex RE` | repo name |
| `--topic T` | has topic T (repeatable; all must match) |
| `--visibility V` | `public`, `private` or `internal` |
| `--language L` | primary language, case-insensitive |
| `--pushed-before DATE` | last push before `YYYY-MM-DD` |
| `--archived` / `--no-archived` | only / exclude archived repos |
| `--forks` / `--no-forks` | only / exclude forks |

Control:

| Flag | Does |
|------|------|
| `--dry-run` | print the plan and exit |
| `-y`, `--yes` | skip the y/N prompt |
| `--all` | required to select every repo in SRC (no names, no filters) |

### Examples

```
# Two named repos
gh repopotamus move pborenstein my-org site-blog site-docs

# Everything matching a glob, minus archived repos, as a dry run
gh repopotamus move pborenstein my-org --match 'site-*' --no-archived --dry-run

# Pipe a filtered list into move
gh repopotamus list pborenstein --pushed-before 2020-01-01 | gh repopotamus move pborenstein attic -

# Names from a file, no prompt
gh repopotamus move my-org pborenstein --from-file repos.txt --yes
```

To see which orgs you can use as SRC or DST, run `gh org list`.

## What happens

1. Lists the repos in SRC and DST.
2. Builds a plan. Repos are skipped, with the reason shown, if the name isn't in SRC or DST already has a repo with that name.
3. Prints the plan and asks `y/N`. If the names came from stdin, the answer is read from the terminal.
4. Transfers one repo at a time. A failure is reported and the rest continue.
5. Prints moved / failed / skipped counts. Exits non-zero if any transfer failed.

GitHub completes a transfer asynchronously. Transferring the same repo again within about 10 seconds fails with 422, so wait a moment before moving something back.

## After moving: GitHub Pages

A moved repo's Pages site changes URL, from `OLD.github.io/repo` to `NEW.github.io/repo` (unless it uses a custom domain). The site needs to be rebuilt, and if the generator bakes in its path (Eleventy `pathPrefix`, Jekyll `baseurl`), update that first.

Check how the site builds:

```
gh api repos/OWNER/REPO/pages -q .build_type
```

| `build_type` | Rebuild with |
|--------------|--------------|
| `workflow`, with a `workflow_dispatch` trigger | `gh workflow run WORKFLOW.yml -R OWNER/REPO` |
| `workflow`, no dispatch trigger | `gh run rerun RUN_ID -R OWNER/REPO` (find the ID with `gh run list -R OWNER/REPO -L 1`) |
| `legacy` (built from a branch) | `gh api -X POST repos/OWNER/REPO/pages/builds` |

## Development

```
go test ./...                     # offline tests
go build -o gh-repopotamus .      # gh picks up the new binary right away
```

Tests that really transfer a repo use the `integration` build tag and need a throwaway repo:

```
REPOPOTAMUS_TEST_REPO=pborenstein/test-repo REPOPOTAMUS_TEST_DEST=documentation-rodeo \
  go test -tags integration -v ./internal/github/
```

Project tracking lives in `docs/`: `CONTEXT.md` for current state, `IMPLEMENTATION.md` for the phase plan, `DECISIONS.md` for design decisions.

## Status

The CLI works. A Bubble Tea TUI is next, followed by precompiled releases.
