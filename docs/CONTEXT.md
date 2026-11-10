---
phase: 2
phase_name: TUI
updated: 2026-11-10
last_commit: 3406f34
last_entry: 8
---

## Current Focus

Cloning works from the CLI (`clone OWNER … --dir`) and the TUI (`c`, `d` for the directory, `--clone-dir`) per DEC-011. Every feature now ships on both surfaces (DEC-012). Real TUI moves and a real clone are confirmed working.

## Active Tasks

- [ ] Scrolling for review screens longer than the terminal (move and clone both truncate; whole-org clones make this likely)
- [ ] Decide whether Phase 2 is done (candidates: `r` to refresh a pane, parallel clones)

## Blockers

None. Real transfers need Philip to run them (the auto-mode classifier blocks Claude).

## Context

- Philip is mainly a CLI user. Build every feature for both CLI and TUI (DEC-012)
- Build with `go build -o gh-repopotamus .`; gh runs the new binary right away (DEC-006). `.go` files use tabs (gofmt)
- Layout: `internal/selection` (filters), `internal/move` and `internal/clone` (plan + `Execute` with injected func and progress callback), `cmd/filters.go` (shared by list/move/clone), `internal/tui`
- Clone: flat `DIR/REPO` via `gh repo clone`; skips existing folders and unknown names; no `--all` guard. The TUI's `c` with nothing selected = all shown repos (no cursor-row fallback)
- `move` needs `--all` with no names/filters (DEC-008); collisions and unknown names are skipped at plan time (DEC-009)
- bubbles' list overruns its height by 1 when paginated, and binds quit to `v`. Both worked around in `internal/tui`
- TUI tests use a harness running each `tea.Cmd`, with fake `listRepos`/`transfer`/`cloneRepo`/`exists`
- `hidden.nogit.dir/` and `xfer`/`xfer.sh` are prior art. Leave them alone

## Next Session

Add scrolling to the review screens, then decide whether Phase 2 is done.
