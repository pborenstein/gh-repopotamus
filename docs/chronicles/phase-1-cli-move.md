# Phase 1: CLI move

## Entry 4: move command (2026-10-29)

**What**: Added `gh repopotamus move SRC DST [REPO...]` with selection, plan table, confirm, `--yes`, `--dry-run`, sequential execution and a summary. `list` takes the same filters.

**Why**: Phase 1 goal. Replaces the hand-built list file that `xfer.sh` needs.

**How**:

- `internal/selection`: `Criteria` and `Select` as pure functions over `[]github.Repo`, plus `ReadNames` for stdin/file input. Table-driven unit tests
- `internal/move`: `NewPlan` and `Execute`, with the transfer function injected so tests use a fake. Meant to be shared with the Phase 2 TUI
- `cmd/filters.go`: filter flags shared by `list` and `move`
- If the names came from stdin, the y/N prompt reads `/dev/tty`
- Checked against GitHub without transferring anything: filtered `list`, the guards, `--dry-run`, stdin piping, and answering "n" at the prompt
- A real `move` round trip has not been run yet: the auto-mode classifier blocked it
- Side notes: `gh org list` already covers listing orgs. Moved Pages sites need a rebuild (`gh workflow run`, `gh run rerun`, or POST `pages/builds` depending on `build_type`); added as a Phase 3 task

**Decisions**:

- DEC-008: Moving every repo needs --all
- DEC-009: Name collisions are skipped at plan time

**Files**: cmd/move.go, cmd/filters.go, cmd/list.go, internal/selection/, internal/move/

## Entry 5: Real round trip passes; Phase 1 closed (2026-10-29)

**What**: Philip ran `move` for real: `test-repo` from pborenstein to documentation-rodeo and back. It works.

**Why**: The last open Phase 1 task. The auto-mode classifier blocks Claude from doing real transfers, so Philip ran it.

**How**: `gh repopotamus move SRC DST test-repo --yes` in each direction.

**Decisions**: None

**Files**: docs only
