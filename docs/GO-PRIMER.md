# Go Primer for This Project

Go concepts mapped to Node equivalents, with pointers to where each one shows up in this repo.

## Runtime and toolchain

| Go | Node analogue | Notes |
|----|---------------|-------|
| `go` (one command) | `node` + `npm` + `tsc` + `eslint --fix` + `jest` | Build, run, test, format and dependency management are all subcommands of `go` |
| Compiled binary | (none) | `go build` produces one native executable with no runtime dependencies. The user doesn't need Go installed |
| Go runtime | V8 | The garbage collector and scheduler are compiled *into* the binary, so there's no separate runtime install |
| `go run .` | `node index.js` | Compiles to a temp dir and runs it. Handy during development |
| `GOOS=linux GOARCH=amd64 go build` | (none) | Cross-compiles for another platform from your Mac |
| Toolchain version | `.nvmrc` / nvm | The `go 1.27.3` line in `go.mod` sets the minimum. Newer Go versions download older toolchains automatically when needed |

Installed here with Homebrew: `/opt/homebrew/bin/go`.

## Dependencies: the `node_modules` question

| Go | Node analogue | Notes |
|----|---------------|-------|
| `go.mod` | `package.json` | Module name and direct and indirect dependencies, with exact versions |
| `go.sum` | `package-lock.json` | Checksums of every dependency. Commit it |
| Module cache `~/go/pkg/mod` | `node_modules` | **Shared across all projects**, read-only and versioned (`go-gh/v2@v2.17.0/`). There's nothing per-project and nothing to gitignore |
| `go get pkg@latest` | `npm install pkg` | Adds or upgrades a dependency in `go.mod` |
| `go mod tidy` | `npm prune` + install | Adds what the code imports and removes what it doesn't. Run it after changing imports |
| `go clean -modcache` | `rm -rf node_modules` | Wipes the shared cache. Rarely needed |
| `// indirect` in go.mod | transitive dep | A dependency of a dependency |
| `vendor/` (optional) | committing `node_modules` | `go mod vendor`. Not used here |

There's no registry like npmjs.com. A module's name *is* its source location (`github.com/spf13/cobra`), and the Go module proxy (proxy.golang.org) caches it. That's also why this module is named `github.com/pborenstein/gh-repopotamus`: it's the URL it will eventually be fetched from.

## Code organization

| Go | Node analogue | Notes |
|----|---------------|-------|
| module | npm package | Everything under `go.mod`. This repo is one module |
| package | a folder of files acting as one module | **A directory is a package.** All `.go` files in `cmd/` are `package cmd` and see each other's identifiers without importing |
| `package main` + `func main()` | `"bin"` entry point | `main.go` here |
| `internal/` directory | (no equivalent) | The compiler stops anyone outside this module from importing it. It's our private API layer |
| Capitalized name (`ListRepos`) | `export` | Visible outside the package |
| lowercase name (`listCmd`) | not exported | Private to the package |
| `func init()` | top-level module code | Runs automatically at startup. Each `cmd/*.go` file uses one to register its subcommand |

## This repo's layout

```
main.go                 entry point: calls cmd.Execute()
cmd/root.go             cobra root command (gh repopotamus)
cmd/list.go             gh repopotamus list OWNER [-l]
cmd/move.go             gh repopotamus move SRC DST [REPO...]
cmd/clone.go            gh repopotamus clone OWNER [REPO...]
cmd/tui.go              gh repopotamus (no args) / tui
internal/github/        API layer: ListRepos (GraphQL), ListOwners and Transfer (REST), Clone (runs gh)
internal/selection/     pure filter logic
internal/move/          plan + Execute, shared by CLI and TUI
internal/clone/         clone plan + Execute, shared by CLI and TUI
internal/tui/           Bubble Tea models: app, pane, owner picker, queue, transfer runner
go.mod / go.sum         dependencies and their checksums
gh-repopotamus          built binary (gitignored)
```

## Libraries in use

| Library | Role | Node analogue |
|---------|------|---------------|
| `github.com/cli/go-gh/v2` | API clients that reuse `gh auth` credentials | `@octokit/*` with token from gh |
| `github.com/spf13/cobra` | Subcommands, flags, help text, shell completion | commander / yargs |
| `charm.land/bubbletea/v2` | TUI framework (Elm architecture) | ink, or a Redux loop |
| `charm.land/bubbles/v2` | Ready-made widgets: list, text input, spinner | ink-select-input etc. |
| `charm.land/lipgloss/v2` | Terminal styling and layout | chalk + a bit of CSS |

## Daily commands

| Task | Command |
|------|---------|
| Build | `go build -o gh-repopotamus .` |
| Run without building | `go run . list tepiton` |
| Run as gh extension | `gh repopotamus list tepiton` (rebuild first, since gh runs the built binary) |
| Format | `gofmt -w .` (there's one canonical style, with no config and no style debates) |
| Lint-ish | `go vet ./...` |
| Test | `go test ./...` (`./...` means "this directory and all below") |
| Update deps | `go get -u ./... && go mod tidy` |

## Language quirks you'll see in the code

- **Errors are return values, not exceptions.** `repos, err := ListRepos(x); if err != nil { return err }`. That pattern is everywhere, on purpose
- **`:=`** declares and assigns. **`=`** assigns to an existing variable
- **Structs, not classes.** Methods attach to types: `func (r Repo) FullName() string`
- **Pointers** (`*struct{...}`) are used where a value may be absent (`nil`). For example, `RepositoryOwner` is nil when the owner doesn't exist
- **JSON mapping** is by field name, case-insensitive: the `IsArchived` field receives `"isArchived"`
- **Import aliases.** `gh "github.com/cli/go-gh/v2"` imports that path under the name `gh`, like `import gh from "..."` in JS. Normally the name is the last path element, but here that's `v2`, so the alias gives it a readable name. `gh.Exec("repo", "clone", ...)` then runs the gh binary and returns its stdout and stderr
- **Tabs for indentation.** `gofmt` enforces it. That's the Go convention, so the 2-space rule doesn't apply to `.go` files

## Bubble Tea in Node terms

Bubble Tea is the Elm architecture, which is close to a Redux store driving a render function.

| Bubble Tea | Node / Redux analogue | In this repo |
|------------|-----------------------|--------------|
| Model (a struct) | Store state | `app` in `internal/tui/app.go`, which holds two `pane`s |
| `Update(msg) (Model, Cmd)` | Reducer: returns the *new* state | Handles key presses, window size, loaded data |
| `View() tea.View` | `render()` | Called after every Update |
| `tea.Msg` | An action | `tea.KeyPressMsg`, `ownersLoadedMsg` |
| `tea.Cmd` (`func() tea.Msg`) | A thunk / side effect | `loadOwners` calls the API; its return value comes back as a Msg |
| `tea.Quit` | `process.exit` after cleanup | Returned on `q` with nothing queued, or after the run |

Two Go details that show up here:

- **Value receivers.** `func (m ownerPicker) Update(...)` gets a *copy* of the model, changes the copy, and returns it. That's why Update returns `m`: like a reducer, nothing is mutated in place. A helper that should modify the model in place takes a pointer instead (`func (p *pane) setRows(...)`)
- **Type switches.** `switch msg := msg.(type) { case tea.KeyPressMsg: ... }` branches on the concrete type behind an interface, like `if (action instanceof X)` but checked by the compiler. `x.(ownerItem)` with `, ok` is a safe cast

Side effects never run inside Update. Update returns a Cmd, Bubble Tea runs it on another goroutine (a lightweight thread), and the result arrives as the next Msg. That's how slow API calls stay off the UI loop.

### Goroutines and channels (`internal/tui/run.go`)

| Go | Node analogue |
|----|---------------|
| `go func() { ... }()` | Starting work without awaiting it, but on a real thread pool, so blocking calls don't freeze anything |
| `ch := make(chan T)` | An async queue one side pushes to and the other awaits; with no buffer, a send waits until someone receives |
| `ch <- v` / `v, ok := <-ch` | `push` / `await next()`. `ok` is false once the channel is closed, like an async iterator finishing |
| `defer close(ch)` | Signals "no more values" when the function returns, however it returns |

The transfers run in one goroutine and send a `progressMsg` per repo. `waitProgress` is a `tea.Cmd` that blocks on the channel; each time a message arrives, Update handles it and returns `waitProgress` again to wait for the next. The goroutine gets its own copy of the plans, so the UI's copy only ever changes inside Update. `go test -race` checks that no two goroutines touch the same memory unsafely.

## How the gh extension is installed

gh runs `gh repopotamus` by finding the `gh-repopotamus` executable inside `~/.local/share/gh/extensions/gh-repopotamus/`. Running `gh extension install .` in this directory makes that path a symlink to the repo. Rebuild the binary and gh picks up the change immediately.
