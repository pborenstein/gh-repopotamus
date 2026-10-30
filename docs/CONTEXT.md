---
phase: 2
phase_name: TUI
updated: 2026-10-29
last_commit: 539e87f
last_entry: 5
---

## Current Focus

Phase 1 is complete: `gh repopotamus move` passed a real round trip. Phase 2 is next: a Bubble Tea TUI that reuses `internal/selection` and `internal/move`.

## Active Tasks

- [ ] `gh repopotamus` with no args (or `tui`) asks for source and destination owners
- [ ] Multi-select repo list (name, visibility, archived, language, last push)
- [ ] Type-to-filter, archived/forks toggles, select all visible, invert
- [ ] Confirm screen showing the plan; progress view ending in a summary

## Blockers

None. Real transfers need Philip to run them (the auto-mode classifier blocks Claude).

## Context

- Go is new to Philip. docs/GO-PRIMER.md maps Go concepts to Node
- Build with `go build -o gh-repopotamus .`; gh runs the new binary right away (DEC-006). `.go` files use tabs (gofmt)
- Layout: `internal/selection` (pure filters), `internal/move` (plan + `Execute` with an injected transfer func and a per-item progress callback), `cmd/filters.go` (shared flags)
- `move` needs `--all` with no names/filters (DEC-008); DST collisions and unknown names are skipped at plan time (DEC-009)
- Owner picker can source orgs from `gh api user/orgs` (same data as `gh org list`) plus the user's own login
- `hidden.nogit.dir/` and `xfer`/`xfer.sh` are prior art. Leave them alone

## Next Session

Create `docs/chronicles/phase-2-tui.md`, add Bubble Tea, and build the owner picker. `move.Execute` runs synchronously; the progress view will need to run it in a goroutine (or a `tea.Cmd` per item) and send each `Item` back as a message.
