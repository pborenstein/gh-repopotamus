# gh-repopotamus

A Go `gh` extension that moves repos between owners in batches. The README covers usage. This file covers how to work in the repo.

## Start here

- `docs/CONTEXT.md` has the current phase, the open tasks and what's next. Read it first.
- `docs/DECISIONS.md` has the design decisions (DEC-NNN). Check it before changing behavior it covers.
- `docs/IMPLEMENTATION.md` has the phase plan. `docs/chronicles/` has a session log for each phase.

## Commands

| Task | Command |
|------|---------|
| Build | `go build -o gh-repopotamus .` (gh runs the new binary right away) |
| Offline tests | `go test ./...` |
| Format / vet | `gofmt -w .` / `go vet ./...` |
| Integration tests | `go test -tags integration -v ./internal/github/` with `REPOPOTAMUS_TEST_REPO` and `REPOPOTAMUS_TEST_DEST` set |

Run `gofmt`, `go vet` and `go test ./...` before you commit.

## Hard rules

- **Never run a real transfer.** That includes `move` without `--dry-run` and the integration tests. Philip runs them. The auto-mode classifier blocks them anyway.
- `move --dry-run` and `list` are read-only, so they're fine to run.
- Leave `hidden.nogit.dir/`, `xfer` and `xfer.sh` alone. They're prior art.
- `.go` files use tabs (gofmt). Ignore the global 2-space rule here.

## Layout

| Path | Role |
|------|------|
| `main.go` | Calls `cmd.Execute()` |
| `cmd/` | cobra commands, one per file, each registering itself in `init()` (DEC-005). `filters.go` holds the selection flags shared by `list` and `move` |
| `internal/github` | go-gh API calls: a paginated GraphQL repo listing and the transfer |
| `internal/selection` | Pure filter logic over repo metadata. No I/O |
| `internal/move` | Builds the plan (with skips for collisions and unknown names, DEC-009) and runs `Execute` with an injected transfer func and a per-item progress callback |
| `internal/tui` | Two-pane browser (DEC-010). `queue.go` is the pure staging logic; `run.go` runs `move.Execute` in a goroutine. Tests drive the app with fakes |

Keep logic in `internal/` so the CLI and the TUI share it. `cmd/` and the TUI should only parse input and render output.

## Working with Philip

- Go is new to Philip. When you introduce an idiom they haven't seen yet (channels, goroutines, `tea.Cmd`, interfaces), explain it in Node terms and add it to `docs/GO-PRIMER.md`.
- Update tracking with the `handoff:session-wrapup` skill: CONTEXT.md, the phase chronicle, and DECISIONS.md for new decisions.
