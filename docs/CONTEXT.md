---
phase: 1
phase_name: CLI move
updated: 2026-10-29
last_commit: 2a75f2e
last_entry: 3
---

## Current Focus

Phase 0 is complete: `gh repopotamus list OWNER [-l]` works, and `Transfer` passed a real round trip. Phase 1 is next: the `move` command.

## Active Tasks

- [ ] `cmd/move.go`: `gh repopotamus move SRC DST [REPO...]`
- [ ] Selection: names, stdin/`--from-file`, `--match`/`--regex`, metadata filters
- [ ] Plan table, y/N confirm, `--yes`, `--dry-run`
- [ ] Execute sequentially, continue on failure, summary, non-zero exit on any failure

## Blockers

None.

## Context

- Go is new to Philip. docs/GO-PRIMER.md maps Go concepts to Node
- Build with `go build -o gh-repopotamus .`; gh runs the new binary right away (DEC-006)
- Integration tests use the `integration` build tag (DEC-007): `REPOPOTAMUS_TEST_REPO=pborenstein/test-repo REPOPOTAMUS_TEST_DEST=documentation-rodeo go test -tags integration -v ./internal/github/`
- A transfer settles asynchronously. Re-transferring the same repo within about 10s gets 422; a name collision also gets 422
- `hidden.nogit.dir/` and `xfer`/`xfer.sh` are prior art. Leave them alone
- `.go` files use tabs (gofmt), not the global 2-space rule

## Next Session

Create `docs/chronicles/phase-1-cli-move.md`, then start `cmd/move.go` with the selection logic. Pure filtering over `[]Repo` is easy to unit test without hitting GitHub.
