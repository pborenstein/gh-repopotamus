---
phase: 0
phase_name: Foundation
updated: 2026-10-29
last_commit: a3d0767
last_entry: 2
---

## Current Focus

The Phase 0 scaffold is done and `gh repopotamus list OWNER [-l]` works. Baseline committed. What's left is testing `Transfer` for real.

## Active Tasks

- [ ] Test `Transfer` with a throwaway repo (create it in one org, move it, verify, delete)
- [ ] Then Phase 1: the `move` command with selection, plan, confirm, execute and summary

## Blockers

None.

## Context

- Go is new to Philip. docs/GO-PRIMER.md maps Go concepts to Node
- Build with `go build -o gh-repopotamus .`. and gh runs the new binary right away (DEC-006)
- The transfer API returns 202 and finishes asynchronously. It fails on a name collision in the target
- `hidden.nogit.dir/` and `xfer`/`xfer.sh` are prior art. Leave them alone
- `.go` files use tabs (gofmt), not the global 2-space rule

## Next Session

Test `Transfer` on a throwaway repo, then start the Phase 1 `move` command in `cmd/move.go`.
