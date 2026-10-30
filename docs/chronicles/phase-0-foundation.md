# Phase 0: Foundation

## Entry 1: Project kickoff (2026-10-29)

**What**: Defined the scope for repopotamus and set up project tracking.

**Why**: `xfer` and `xfer.sh` work, but a batch move first needs a hand-built list file. The goal is a CLI and a TUI that select repos directly.

**How**:

- Reviewed the prior art: `xfer` (one repo) and `xfer.sh` (list file, hardcoded owners, stops on failure)
- Chose the stack: a Go gh extension using go-gh and Bubble Tea
- Defined the CLI selection model: names, stdin/file, glob/regex and filters, combined
- Defined the safety model: plan, then confirm, with `--yes` and `--dry-run`. Batches continue on failure and print a summary

**Decisions**:

- DEC-001: Go gh extension
- DEC-002: CLI selection sources combine
- DEC-003: Confirm before transferring
- DEC-004: Continue on failure, summarize at end

**Files**: docs/CONTEXT.md, docs/IMPLEMENTATION.md, docs/DECISIONS.md

## Entry 2: Scaffold and list command (2026-10-29)

**What**: Created the Go module and the cobra command tree, plus the API layer with `ListRepos` and `Transfer`. `gh repopotamus list` works.

**Why**: Phase 0 foundation. The selection logic in Phase 1 depends on the repo metadata from `ListRepos`.

**How**:

- `go mod init github.com/pborenstein/gh-repopotamus`; dependencies are go-gh v2.17.0 and cobra v1.10.2
- `ListRepos` uses GraphQL `repositoryOwner`, with `ownerAffiliations: [OWNER]` and cursor pagination
- `Transfer` uses REST POST. It hasn't been run against a real repo yet
- Renamed the project directory `repopotamus/` to `gh-repopotamus/`, because `gh extension install .` rejects directory names without `gh-`
- Added docs/GO-PRIMER.md (Go concepts mapped to Node)

**Decisions**:

- DEC-005: cobra
- DEC-006: Everything named gh-repopotamus

**Files**: main.go, cmd/root.go, cmd/list.go, internal/github/github.go, docs/GO-PRIMER.md
