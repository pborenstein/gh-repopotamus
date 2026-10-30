---
phase: 1
phase_name: CLI move
updated: 2026-10-29
last_commit: ff32bc1
last_entry: 4
---

## Current Focus

`gh repopotamus move SRC DST [REPO...]` is written and checked against GitHub with everything except a real transfer. Phase 1 closes once a real round trip passes.

## Active Tasks

- [x] `cmd/move.go`, selection, plan table, confirm, `--yes`, `--dry-run`, summary, exit code
- [x] `list SRC [filters]` shares the filter flags
- [ ] Real `move` round trip with `test-repo` (blocked for Claude by the auto-mode classifier; Philip runs it)

## Blockers

None. Real transfers need Philip to run them or add a permission rule.

## Context

- Go is new to Philip. docs/GO-PRIMER.md maps Go concepts to Node
- Build with `go build -o gh-repopotamus .`; gh runs the new binary right away (DEC-006). `.go` files use tabs (gofmt)
- Layout: `internal/selection` (pure filters), `internal/move` (plan + execute, transfer injected), `cmd/filters.go` (shared flags). Phase 2 TUI reuses the two internal packages
- `move` needs `--all` with no names/filters (DEC-008); DST name collisions and unknown names are skipped at plan time (DEC-009)
- A transfer settles asynchronously; re-transferring within ~10s gets 422. Integration tests use the `integration` tag (DEC-007)
- Moved repos with Pages need a rebuild; the approach is noted as a Phase 3 task. `gh org list` already lists orgs
- `hidden.nogit.dir/` and `xfer`/`xfer.sh` are prior art. Leave them alone

## Next Session

Run the round trip: `gh repopotamus move pborenstein documentation-rodeo test-repo --yes`, then move it back (once immediately to see the 422 failure path, then again after a minute). If it passes, close Phase 1 and start the Phase 2 TUI with Bubble Tea.
