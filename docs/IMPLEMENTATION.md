# Implementation

repopotamus is a `gh` extension (`gh repopotamus`) that transfers one or more repos from one GitHub owner (org or user) to another. It has a CLI mode and a Bubble Tea TUI mode.

## Phase Overview

| Phase | Name | Status | Chronicle |
|-------|------|--------|-----------|
| 0 | Foundation | Complete | [phase-0-foundation.md](chronicles/phase-0-foundation.md) |
| 1 | CLI move | Complete | [phase-1-cli-move.md](chronicles/phase-1-cli-move.md) |
| 2 | TUI | Next | - |
| 3 | Polish and release | Planned | - |

## Prior Art

Bash scripts in the repo root. They stay until repopotamus replaces them.

| File | Does |
|------|------|
| `xfer` | `xfer owner/repo newowner`. Transfers one repo with `gh api POST repos/{repo}/transfer` |
| `xfer.sh` | Reads repo names from a `sites` file. The source and destination owners are hardcoded. Stops at the first failure |

Pain point: you have to build the list file by hand before a batch move.

## Phase 0: Foundation (complete)

Goal: a gh extension scaffold that authenticates through gh and can list a source owner's repos with their metadata.

### Tasks

- [x] `git init`, `.gitignore`, `go mod init github.com/pborenstein/gh-repopotamus`
- [x] Scaffold the extension layout (`main.go`, `cmd/`, `internal/github/`)
- [x] Wire up the go-gh REST and GraphQL clients, with auth taken from gh
- [x] `internal/github`: `ListRepos(owner)` returns name, visibility, archived, fork, topics, language and pushedAt
  - GraphQL `repositoryOwner(login:)` works for orgs and users alike
  - Paginated (100 per page). Tested against tepiton, mimeo-sites and pborenstein (70 repos)
- [x] `internal/github`: `Transfer(owner, repo, newOwner)` sends `POST /repos/{owner}/{repo}/transfer`
- [x] CLI framework: cobra (DEC-005)
- [x] `gh repopotamus` works locally (`gh extension install .`, DEC-006)
- [x] `gh repopotamus list OWNER [-l]`
- [x] Test `Transfer` against a throwaway repo: round trip pborenstein (user) to documentation-rodeo (org) and back passes (`transfer_integration_test.go`, `-tags integration`)
- [x] First commit (baseline)

### Notes

- The transfer endpoint returns 202 and finishes asynchronously. The repo can still answer at the old path for a moment afterwards
- Transferring into an org requires the caller to have permission to create repos there
- The repo resolves at its new owner before GitHub finishes the transfer. Transferring the *same* repo again within about 5-10 seconds gets 422 "Repository.data is invalid". Batches move each repo once, so they aren't affected
- Transfer fails if the target already has a repo with the same name. `new_name` is an optional field and could be offered later

## Phase 1: CLI move (complete)

Command shape (draft):

```
gh repopotamus move SRC DST [REPO...] [flags]
```

Selection. These combine: explicit names are unioned with stdin/file, and the result is then narrowed by `--match` and the filters.

| Input | Flag |
|-------|------|
| Explicit names | positional `REPO...` |
| stdin / file | `-` as positional, or `--from-file PATH` |
| Glob / regex | `--match 'site-*'`, `--regex '^site-'` |
| Filters | `--topic`, `--archived`/`--no-archived`, `--visibility public\|private\|internal`, `--language`, `--pushed-before DATE`, `--forks`/`--no-forks` |

Execution:

- [x] Resolve the selection into a plan: a table of repos that shows source and destination
- [x] Confirm with y/N. `--yes` skips the prompt. `--dry-run` prints the plan and exits
- [x] Transfer sequentially and keep going on failure
- [x] Print a summary at the end: moved, failed (with the API error message), skipped
- [x] Exit non-zero if anything failed
- [x] `gh repopotamus list SRC [filters]` prints the names the selection matched (useful for piping)
- [x] Real `move` round trip with `test-repo` (pborenstein to documentation-rodeo and back), run by Philip

## Phase 2: TUI (current)

- [ ] `gh repopotamus` with no args (or `tui`) asks for the source and destination owners
- [ ] Multi-select list of source repos showing name, visibility, archived, language and last push
- [ ] Type-to-filter, plus toggles for archived/forks, select all visible, and invert
- [ ] Confirm screen that shows the plan
- [ ] Progress view with per-repo status, ending in a summary
- [ ] Reuse the same `plan` and `execute` code as the CLI

## Phase 3: Polish and release

- [ ] `new_name` support on collisions (maybe)
- [ ] Owner completion from `gh api user/orgs` (same data as `gh org list`)
- [ ] Post-move Pages rebuild: pick by `build_type` (`workflow` -> `gh workflow run` or rerun the last run; `legacy` -> POST `pages/builds`). Warn that baseurl/pathPrefix may need changing
- [ ] README, usage examples
- [ ] goreleaser / `gh extension` precompiled release workflow
- [ ] Retire `xfer` and `xfer.sh`
