# Decisions

Architectural decisions for this project. Search with `grep -i "keyword" docs/DECISIONS.md`.

## Active Decisions

### DEC-001: Go gh extension (2026-10-29)

**Status**: Active

**Context**: The existing scripts depend on gh for auth. The new tool needs both a CLI and a TUI.

**Decision**: Build it as a Go `gh` extension. go-gh supplies the auth and API clients, and Bubble Tea supplies the TUI.

**Alternatives considered**: Python + Textual (shells out to gh or reads `gh auth token`); bash + gum/fzf (lightest, but the TUI is limited).

**Consequences**: It ships as one binary, installed with `gh extension install`, and needs no token handling of its own. Go is required to build it.

---

### DEC-002: CLI selection sources combine (2026-10-29)

**Status**: Active

**Context**: Building a list file before every batch move is the main pain point.

**Decision**: Support all four selection sources: explicit names, stdin/file, glob/regex and metadata filters. Names from the list sources are unioned, then narrowed by the match and the filters.

**Alternatives considered**: Only one selection method.

**Consequences**: The selection logic needs the owner's full repo list with metadata, which means one paginated GraphQL query per run.

---

### DEC-003: Confirm before transferring (2026-10-29)

**Status**: Active

**Context**: A transfer is outward-facing and slow to undo: it changes URLs and team access in the target org.

**Decision**: Always show the plan and prompt y/N. `--yes` skips the prompt and `--dry-run` only prints the plan.

**Alternatives considered**: Dry-run by default with `--execute`; no prompt (the `xfer.sh` behavior).

**Consequences**: Scripted use needs `--yes`.

---

### DEC-004: Continue on failure, summarize at end (2026-10-29)

**Status**: Active

**Context**: `xfer.sh` stops at the first failure, which leaves you to work out where the batch ended.

**Decision**: Attempt every repo in the plan, then report moved, failed (with the reason) and skipped. Exit non-zero if any repo failed.

**Alternatives considered**: Stop immediately.

**Consequences**: A failure such as a name collision doesn't block the other repos.

---

### DEC-005: cobra for the command tree (2026-10-29)

**Status**: Active

**Context**: The tool needs subcommands (`list`, `move`, later `tui`), lots of flags, and help text.

**Decision**: Use spf13/cobra. Each subcommand lives in its own `cmd/*.go` file and registers itself in `init()`.

**Alternatives considered**: Standard library `flag` (no subcommands, so they'd have to be hand-rolled); urfave/cli.

**Consequences**: gh itself uses cobra, so help output and flag conventions match. Shell completion comes free.

---

### DEC-006: Directory, module and binary all named gh-repopotamus (2026-10-29)

**Status**: Active

**Context**: `gh extension install .` requires the directory to be named `gh-*`. The project lived in `repopotamus/`.

**Decision**: Rename the project directory to `gh-repopotamus`, to match the Go module path, the binary name and the future GitHub repo name. Install with plain `gh extension install .`.

**Alternatives considered**: Keep `repopotamus/` and hand-symlink it into `~/.local/share/gh/extensions/` (tried briefly, dropped as an extra indirection).

**Consequences**: Directory, module, binary and repo use one name. The binary still has to be rebuilt after changes.

---

### DEC-007: Integration tests behind a build tag (2026-10-29)

**Status**: Active

**Context**: Testing `Transfer` means really moving a repo on GitHub, which is outward-facing and slow.

**Decision**: Tests that call GitHub carry `//go:build integration` and take their targets from env vars (`REPOPOTAMUS_TEST_REPO`, `REPOPOTAMUS_TEST_DEST`). They skip if the env vars are unset. Plain `go test ./...` runs only offline tests.

**Alternatives considered**: A throwaway script in scratch space (not repeatable); mocking the API with go-gh's gock support (doesn't catch real API behavior such as the 422 settle window).

**Consequences**: Real-API checks have to be run on purpose with `-tags integration`. Selection and plan logic in Phase 1 should be written as pure functions so they can be unit tested offline.

---

### DEC-008: Moving every repo needs --all (2026-10-29)

**Status**: Active

**Context**: With no names, the filters apply to every repo in SRC. `move SRC DST` with no names and no filters would select the whole account.

**Decision**: If there are no names and no filters, `move` refuses unless `--all` is passed. `list` has no such guard.

**Alternatives considered**: Rely on the y/N prompt alone (one stray `-y` empties an account).

**Consequences**: An extra flag in the rare case where moving everything is intended.

---

### DEC-009: Name collisions are skipped at plan time (2026-10-29)

**Status**: Active

**Context**: A transfer into an owner that already has a repo with the same name fails with 422. The same 422 also covers the settle window (Entry 3), so the error code alone doesn't say which one happened.

**Decision**: `move` lists DST's repos too and marks collisions as skipped in the plan, with the reason. Names not found in SRC are also skipped in the plan, not treated as errors.

**Alternatives considered**: Let the transfer fail and report the 422 (vaguer message, and it counts as a failure).

**Consequences**: One more GraphQL listing per run. A 422 during execution now usually means the settle window or a race.
