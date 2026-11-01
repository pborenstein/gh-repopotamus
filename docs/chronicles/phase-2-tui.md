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
