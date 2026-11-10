# Phase 2: TUI

## Entry 6: Bubble Tea and the owner picker (2026-10-31)

**What**: Added Bubble Tea v2, bubbles and lipgloss. `gh repopotamus` with no args (or `tui`) opens a two-step owner picker: source, then destination. For now it prints `SRC -> DST` and exits.

**Why**: First Phase 2 task. The repo list needs both owners before it can load anything.

**How**:

- Charm's v2 libraries moved to `charm.land/...` import paths. `View()` returns a `tea.View` struct (alt screen is a field on it), and key presses arrive as `tea.KeyPressMsg`
- `github.ListOwners`: REST `user` plus paginated `user/orgs`. The owners load in a `tea.Cmd`, off the UI loop
- `internal/tui/owners.go`: a bubbles `list` with type-to-filter. Enter picks, esc on the destination step goes back. The source owner is left out of the destination list
- `showOwners` resets the filter and the cursor whenever the list changes. Without it, the cursor kept its row from the source step (caught by `TestPickFlow`)
- Root command gets `RunE = runTUI` and `cobra.NoArgs`. Subcommands still dispatch normally
- Added a Bubble Tea section to docs/GO-PRIMER.md and a CLAUDE.md

**Decisions**: None

**Files**: CLAUDE.md, cmd/tui.go, internal/github/github.go, internal/tui/, docs/GO-PRIMER.md, go.mod, go.sum

## Entry 7: Two-pane browser (2026-10-31)

**What**: Replaced the owners-then-repos flow with a two-pane browser modeled on Font/DA Mover. Each pane shows one owner's repos. Moves are queued in either direction and confirmed on a review screen before anything transfers.

**Why**: The linear flow made you know what was in each org before starting. Philip wanted to browse first.

**How**:

- First built a single multi-select repo list after the owner picker. Philip tried the picker and asked for browsing instead, so the list became a pane
- `internal/tui/queue.go`: pure staging logic. `project` gives an owner's repos after the queue runs; `stage` refuses collisions, cancels moves back to origin, and re-routes to a third owner without chaining; `plans` groups by from/to and calls `move.NewPlan`
- `internal/tui/pane.go`: one owner's list, selection, toggles. Columns drop in priority order when the pane is narrow
- `internal/tui/run.go`: `move.Execute` runs in a goroutine on its own copy of the plans and reports over a channel; a `tea.Cmd` waits on the channel and Update re-issues it after each report
- `cmd/tui.go` prints the results after exit, because the alternate screen erases them. Exits non-zero if any transfer failed
- Bugs caught by tests: the cursor stayed on its row when the owner list changed, and stayed past the end after rows moved out of a pane. Also found that bubbles' list binds quit to `v`; rebound to `q`
- First live run showed stale lines in the left pane: bubbles' list renders one line over its height once it paginates, so the frame overran the terminal and the redraw drifted. Panes give the list `height-1` and clamp with `MaxHeight`; `TestBrowserFitsScreen` checks the frame height
- Tests drive the app with a harness that runs each returned `tea.Cmd` and feeds the message back, with fake `listRepos` and `transfer`. `go test -race` is clean

**Decisions**:

- DEC-010: TUI is a two-pane browser with a queue

**Files**: internal/tui/, cmd/tui.go, docs/DECISIONS.md, docs/IMPLEMENTATION.md, docs/GO-PRIMER.md

## Entry 8: Cloning, from the CLI and the TUI (2026-11-10)

**What**: Clone an owner's repos locally. In the TUI, `c` clones the selection or every shown repo, and `d` changes the destination. On the CLI, `gh repopotamus clone OWNER [REPO...] [filters] --dir DIR`. The TUI also takes `--clone-dir`.

**Why**: Philip wanted local copies: a chosen few from mixed owners, everything from cohesive ones. They also confirmed the real TUI move works, and that the stale-lines fix holds.

**How**:

- `internal/clone`: `NewPlan` (skips existing folders), `AddMissing`, `ResolveDir` (`~`, relative to the start directory, refuses files), and `Execute` with an injected clone func
- `github.Clone` runs `gh repo clone` through `gh.Exec` and reports gh's stderr on failure
- TUI: modes for clone review, cloning and done; a `textinput` for the directory; `startClones` mirrors `startTransfers`. A finished clone returns to the browser
- First built TUI-only. Philip asked for CLI parity, so `cmd/clone.go` reuses `gatherNames`, the shared filters and `confirm`. Dry runs against the real account checked the skips and path resolution
- Tests: clone plan and execute, ResolveDir table, and TUI harness tests for clone-all, selected, esc, and changing the directory

**Decisions**:

- DEC-011: Cloning repos locally
- DEC-012: Everything works from the CLI and the TUI

**Files**: internal/clone/, cmd/clone.go, cmd/tui.go, internal/github/github.go, internal/tui/, README.md
