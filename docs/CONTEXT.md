---
phase: 2
phase_name: TUI
updated: 2026-11-02
last_commit: 539e87f
last_entry: 7
---

## Current Focus

Phase 2 TUI is a two-pane browser modeled on Font/DA Mover (DEC-010). Built and tested with fakes. Philip has browsed it live; a pagination overflow (stale lines) was fixed. It still needs a real move.

## Active Tasks

- [ ] Philip confirms the stale-lines fix (help line visible under the status line)
- [ ] Philip runs a real move from the TUI (queue, review, `y`)
- [ ] Scrolling for a review longer than the screen (it truncates now)

## Blockers

None. Real transfers need Philip to run them (the auto-mode classifier blocks Claude).

## Context

- Go is new to Philip. docs/GO-PRIMER.md maps Go concepts to Node
- Build with `go build -o gh-repopotamus .`; gh runs the new binary right away (DEC-006). `.go` files use tabs (gofmt)
- Layout: `internal/selection` (pure filters), `internal/move` (plan + `Execute` with an injected transfer func and a per-item progress callback), `cmd/filters.go` (shared flags), `internal/tui` (app, pane, queue, owner picker, transfer runner)
- `move` needs `--all` with no names/filters (DEC-008); DST collisions and unknown names are skipped at plan time (DEC-009)
- `github.ListOwners` = the user's login plus `user/orgs` (same data as `gh org list`)
- bubbles' list overruns its height by 1 when paginated, and binds quit to `v`. Both are worked around in `internal/tui`
- TUI tests use a harness that runs each `tea.Cmd` and feeds the message back, with fake `listRepos`/`transfer`
- `hidden.nogit.dir/` and `xfer`/`xfer.sh` are prior art. Leave them alone

## Next Session

Hear back from Philip's real TUI run and fix what turns up. Then review scrolling, and decide whether Phase 2 is done or needs more (e.g. refreshing a pane from the API with `r`).
