// Package tui is the interactive front end: a two-pane browser in the style
// of the classic Mac Font/DA Mover. It shares plan and execute logic with the
// CLI and only handles input and rendering.
package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pborenstein/gh-repopotamus/internal/clone"
	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/move"
)

// newList returns a list with the settings every screen shares. The stock
// keymap binds quit to "v", so it is rebound to "q" here.
func newList(d list.ItemDelegate) list.Model {
	l := list.New(nil, d, 0, 0)
	l.KeyMap.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	return l
}

type mode int

const (
	modeLoading     mode = iota // waiting for the owner list
	modeBrowse                  // two panes
	modePickOwner               // owner list open in the active pane
	modeReview                  // queued moves shown as plans, waiting for y
	modeRunning                 // transfers in progress
	modeDone                    // transfers finished
	modeCloneReview             // clone plan shown, waiting for y
	modeCloning                 // clones in progress
	modeCloneDone               // clones finished; any key returns to browse
)

// Result is what happened. Ran is false if the user quit without confirming.
type Result struct {
	Ran   bool
	Plans []move.Plan
}

type app struct {
	mode          mode
	width, height int

	owners []github.Owner
	cache  map[string][]github.Repo
	panes  [2]pane
	active int
	picker ownerPicker
	queue  queue
	status string

	plans    []move.Plan
	progress chan progressMsg
	aborting bool

	clonePlan     clone.Plan
	cloneRepos    []github.Repo // what clonePlan was built from
	cloneProgress chan cloneProgressMsg
	dirInput      textinput.Model
	editingDir    bool

	listRepos func(string) ([]github.Repo, error)
	transfer  move.TransferFunc
	startDir  string // where gh repopotamus was started; relative paths start here
	cloneDir  string // where clones go; starts as startDir
	exists    func(string) bool
	cloneRepo clone.CloneFunc
	err       error
}

func newApp() app {
	// Clones go into the directory gh repopotamus was started from.
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	in := textinput.New()
	in.Prompt = ""
	return app{
		dirInput:  in,
		startDir:  dir,
		cache:     map[string][]github.Repo{},
		panes:     [2]pane{newPane(), newPane()},
		picker:    newOwnerPicker(),
		listRepos: github.ListRepos,
		transfer:  github.Transfer,
		cloneDir:  dir,
		exists:    clone.Exists,
		cloneRepo: github.Clone,
	}
}

func (a app) Init() tea.Cmd {
	return loadOwners
}

// errMsg ends the program with an error, e.g. a failed API call.
type errMsg struct{ err error }

// reposLoadedMsg carries one owner's repos, or the error loading them.
type reposLoadedMsg struct {
	owner string
	repos []github.Repo
	err   error
}

func send(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func (a app) loadRepos(owner string) tea.Cmd {
	list := a.listRepos
	return func() tea.Msg {
		repos, err := list(owner)
		return reposLoadedMsg{owner, repos, err}
	}
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.resize()
		return a, nil

	case errMsg:
		a.err = msg.err
		return a, tea.Quit

	case ownersLoadedMsg:
		a.owners = msg.owners
		// Open yourself on the left, then ask what to open on the right.
		cmd := a.open(0, a.owners[0].Login)
		a.setActive(1)
		a.openPicker()
		return a, cmd

	case reposLoadedMsg:
		if msg.err != nil {
			a.status = fmt.Sprintf("Could not load %s: %v", msg.owner, msg.err)
			for i := range a.panes {
				if a.panes[i].owner == msg.owner {
					a.panes[i].owner = ""
					a.panes[i].loading = false
					a.panes[i].setTitle()
				}
			}
			return a, nil
		}
		a.cache[msg.owner] = msg.repos
		return a, a.refresh()

	case ownerChosenMsg:
		a.mode = modeBrowse
		return a, a.open(a.active, msg.login)

	case ownerCancelMsg:
		a.mode = modeBrowse
		return a, nil

	case progressMsg:
		apply(a.plans, msg)
		return a, waitProgress(a.progress)

	case transfersDoneMsg:
		a.mode = modeDone
		return a, nil

	case cloneProgressMsg:
		applyClone(&a.clonePlan, msg)
		return a, waitClone(a.cloneProgress)

	case clonesDoneMsg:
		a.mode = modeCloneDone
		a.aborting = false
		a.status = ""
		return a, nil

	case tea.KeyPressMsg:
		return a.key(msg)
	}

	// Anything else (cursor blink, status timeouts) goes to the widget that
	// has focus.
	var cmd tea.Cmd
	switch a.mode {
	case modePickOwner:
		a.picker, cmd = a.picker.Update(msg)
	case modeBrowse:
		p := &a.panes[a.active]
		p.list, cmd = p.list.Update(msg)
	case modeCloneReview:
		if a.editingDir {
			a.dirInput, cmd = a.dirInput.Update(msg)
		}
	}
	return a, cmd
}

func (a app) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		if (a.mode == modeRunning || a.mode == modeCloning) && !a.aborting {
			a.aborting = true
			what := "Transfers are"
			if a.mode == modeCloning {
				what = "Clones are"
			}
			a.status = what + " running. Press ctrl+c again to abort."
			return a, nil
		}
		return a, tea.Quit
	}

	switch a.mode {
	case modePickOwner:
		var cmd tea.Cmd
		a.picker, cmd = a.picker.Update(msg)
		return a, cmd

	case modeReview:
		switch k {
		case "y":
			if countPending(a.plans) == 0 {
				return a, nil
			}
			a.mode = modeRunning
			a.progress = startTransfers(a.plans, a.transfer)
			return a, waitProgress(a.progress)
		case "esc", "n":
			a.mode = modeBrowse
		}
		return a, nil

	case modeDone:
		return a, tea.Quit

	case modeCloneReview:
		if a.editingDir {
			return a.dirKey(msg)
		}
		switch k {
		case "d":
			a.editingDir = true
			a.status = ""
			a.dirInput.SetValue(a.cloneDir)
			a.dirInput.CursorEnd()
			return a, a.dirInput.Focus()
		case "y":
			if a.clonePlan.Count(clone.Pending) == 0 {
				return a, nil
			}
			a.mode = modeCloning
			a.cloneProgress = startClones(a.clonePlan, a.cloneRepo)
			return a, waitClone(a.cloneProgress)
		case "esc", "n":
			a.mode = modeBrowse
		}
		return a, nil

	case modeCloneDone:
		// Cloning changes nothing on GitHub, so the browser is still current.
		a.mode = modeBrowse
		a.status = cloneSummary(a.clonePlan)
		return a, nil

	case modeBrowse:
		return a.browseKey(msg)
	}
	return a, nil
}

func (a app) browseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := &a.panes[a.active]
	if p.filtering() {
		var cmd tea.Cmd
		p.list, cmd = p.list.Update(msg)
		return a, cmd
	}
	a.status = ""

	switch msg.String() {
	case "tab", "shift+tab":
		a.setActive(1 - a.active)
	case "o":
		a.openPicker()
	case ">", "<", "m":
		return a, a.stage()
	case "c":
		a.reviewClones()
	case "space":
		p.toggleCurrent()
	case "a":
		p.selectAllShown()
	case "i":
		p.invertShown()
	case "A":
		p.hideArchived = !p.hideArchived
		return a, p.setRows(p.rows)
	case "F":
		p.hideForks = !p.hideForks
		return a, p.setRows(p.rows)
	case "q":
		if len(a.queue) == 0 {
			return a, tea.Quit
		}
		a.plans = a.queue.plans(a.cache)
		a.mode = modeReview
	default:
		var cmd tea.Cmd
		p.list, cmd = p.list.Update(msg)
		return a, cmd
	}
	return a, nil
}

// open points pane i at owner, loading its repos unless they're cached.
func (a *app) open(i int, owner string) tea.Cmd {
	a.panes[i].open(owner)
	if _, ok := a.cache[owner]; ok {
		return a.refresh()
	}
	return a.loadRepos(owner)
}

// refresh redraws every pane whose owner's repos are loaded.
func (a *app) refresh() tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.panes {
		p := &a.panes[i]
		if repos, ok := a.cache[p.owner]; ok && p.owner != "" {
			cmds = append(cmds, p.setRows(a.queue.project(p.owner, repos)))
		}
	}
	return tea.Batch(cmds...)
}

func (a *app) openPicker() {
	a.picker.show(a.owners, a.panes[1-a.active].owner, a.panes[a.active].owner)
	a.mode = modePickOwner
}

func (a *app) setActive(i int) {
	a.active = i
	for j := range a.panes {
		bg := lipgloss.Color("240")
		if j == i {
			bg = accent
		}
		a.panes[j].list.Styles.Title = a.panes[j].list.Styles.Title.Background(bg)
	}
}

// stage queues the active pane's chosen rows to move to the other pane.
func (a *app) stage() tea.Cmd {
	src, dst := &a.panes[a.active], &a.panes[1-a.active]
	switch {
	case dst.owner == "":
		a.status = "Open an owner in the other pane first (tab, then o)."
		return nil
	case src.loading || dst.loading:
		a.status = "Still loading."
		return nil
	}
	rows := src.chosen()
	if len(rows) == 0 {
		a.status = "Nothing to move."
		return nil
	}
	var queued, cancelled int
	var skipped []string
	for _, r := range rows {
		destRows := a.queue.project(dst.owner, a.cache[dst.owner])
		q, err := a.queue.stage(r, dst.owner, destRows)
		if err != nil {
			skipped = append(skipped, r.repo.Name)
			continue
		}
		if len(q) < len(a.queue) {
			cancelled++
		} else {
			queued++
		}
		a.queue = q
	}
	clear(src.selected)
	cmd := a.refresh()

	var parts []string
	if queued > 0 {
		parts = append(parts, fmt.Sprintf("queued %d", queued))
	}
	if cancelled > 0 {
		parts = append(parts, fmt.Sprintf("returned %d", cancelled))
	}
	if len(skipped) > 0 {
		parts = append(parts, fmt.Sprintf("skipped %s (name taken in %s)", strings.Join(skipped, ", "), dst.owner))
	}
	a.status = strings.Join(parts, "; ")
	return cmd
}

// dirKey handles keys while the clone directory is being edited. enter
// re-plans against the new directory, since which folders already exist
// depends on it.
func (a app) dirKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		dir, err := clone.ResolveDir(a.dirInput.Value(), a.startDir)
		if err != nil {
			a.status = err.Error()
			return a, nil
		}
		a.cloneDir = dir
		a.clonePlan = clone.NewPlan(dir, a.cloneRepos, a.exists)
	case "esc":
	default:
		var cmd tea.Cmd
		a.dirInput, cmd = a.dirInput.Update(msg)
		return a, cmd
	}
	a.editingDir = false
	a.status = ""
	a.dirInput.Blur()
	return a, nil
}

// reviewClones plans cloning the active pane's selection, or all its shown
// repos, and opens the clone review.
func (a *app) reviewClones() {
	p := &a.panes[a.active]
	switch {
	case p.owner == "":
		a.status = "Open an owner first (o)."
		return
	case p.loading:
		a.status = "Still loading."
		return
	}
	repos := p.cloneTargets()
	if len(repos) == 0 {
		a.status = "Nothing to clone."
		return
	}
	a.cloneRepos = repos
	a.clonePlan = clone.NewPlan(a.cloneDir, repos, a.exists)
	a.mode = modeCloneReview
}

// Layout: two bordered panes with a 4-column gutter, then a status line and
// a help line.
const gutter = 4

func (a *app) resize() {
	outerW := (a.width - gutter) / 2
	outerH := a.height - 2
	innerW, innerH := max(outerW-2, 10), max(outerH-2, 3)
	for i := range a.panes {
		a.panes[i].setSize(innerW, innerH)
	}
	a.picker.list.SetSize(innerW, innerH)
}

func countPending(plans []move.Plan) int {
	n := 0
	for _, p := range plans {
		n += p.Count(move.Pending)
	}
	return n
}

func (a app) View() tea.View {
	var content string
	switch a.mode {
	case modeLoading:
		content = "Loading owners..."
	case modeBrowse, modePickOwner:
		content = a.browserView()
	case modeCloneReview, modeCloning, modeCloneDone:
		content = a.cloneView()
	default:
		content = a.plansView()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

var (
	activeBorder   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent)
	inactiveBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	helpStyle      = lipgloss.NewStyle().Faint(true)
)

func (a app) browserView() string {
	outerW := (a.width - gutter) / 2
	outerH := a.height - 2

	var boxes [2]string
	for i, p := range a.panes {
		style := inactiveBorder
		if i == a.active {
			style = activeBorder
		}
		body := p.View()
		if i == a.active && a.mode == modePickOwner {
			body = a.picker.View()
		}
		// MaxHeight cuts anything taller than the box. A frame taller than
		// the terminal throws off the redraw and leaves stale lines behind.
		boxes[i] = style.Width(outerW).Height(outerH).MaxHeight(outerH).Render(body)
	}

	arrow := " >> "
	if a.active == 1 {
		arrow = " << "
	}
	mid := lipgloss.NewStyle().Width(gutter).Height(outerH).
		AlignVertical(lipgloss.Center).Foreground(accent).Bold(true).Render(arrow)

	status := a.status
	if status == "" {
		status = queueSummary(a.queue)
	}
	help := "tab pane · o owner · space select · a all · i invert · A/F hide archived/forks · / filter · > move · c clone · q review/quit"
	if a.mode == modePickOwner {
		help = "enter open · / filter · esc cancel"
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, boxes[0], mid, boxes[1]) + "\n" +
		status + "\n" + helpStyle.Render(help)
}

func queueSummary(q queue) string {
	if len(q) == 0 {
		return "Nothing queued."
	}
	return fmt.Sprintf("%d move(s) queued. Press q to review.", len(q))
}

var (
	statusStyles = map[move.Status]lipgloss.Style{
		move.Pending: lipgloss.NewStyle().Faint(true),
		move.Moved:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		move.Failed:  lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		move.Skipped: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	}
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
)

// plansView is the review, progress and summary screen.
func (a app) plansView() string {
	var lines []string
	for _, p := range a.plans {
		lines = append(lines, headerStyle.Render(fmt.Sprintf("%s -> %s", p.Src, p.Dst)))
		for _, it := range p.Items {
			label := it.Status.String()
			if a.mode == modeReview && it.Status == move.Pending {
				label = "move"
			}
			line := fmt.Sprintf("  %-8s %s", statusStyles[it.Status].Render(fmt.Sprintf("%-8s", label)), it.Name)
			if it.Reason != "" {
				line += "  " + helpStyle.Render(it.Reason)
			}
			if it.Err != nil {
				line += "  " + helpStyle.Render(it.Err.Error())
			}
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}

	lines = a.fit(lines)

	var footer, help string
	switch a.mode {
	case modeReview:
		pending := countPending(a.plans)
		skipped := 0
		for _, p := range a.plans {
			skipped += p.Count(move.Skipped)
		}
		footer = fmt.Sprintf("%d to transfer, %d skipped.", pending, skipped)
		help = "y transfer · esc back to browser · ctrl+c quit without moving"
		if pending == 0 {
			help = "esc back to browser · ctrl+c quit"
		}
	case modeRunning:
		done := 0
		total := 0
		for _, p := range a.plans {
			done += p.Count(move.Moved) + p.Count(move.Failed)
			total += p.Count(move.Moved) + p.Count(move.Failed) + p.Count(move.Pending)
		}
		footer = fmt.Sprintf("Transferring %d/%d...", done, total)
		if a.status != "" {
			footer = a.status
		}
		help = "ctrl+c abort"
	case modeDone:
		footer = summary(a.plans)
		help = "any key to exit"
	}
	return strings.Join(lines, "\n") + "\n" + footer + "\n" + helpStyle.Render(help)
}

// fit cuts lines to leave room for a footer and help line under them.
func (a app) fit(lines []string) []string {
	if room := a.height - 3; room > 0 && len(lines) > room {
		more := len(lines) - room + 1
		lines = append(lines[:room-1], helpStyle.Render(fmt.Sprintf("... %d more lines", more)))
	}
	return lines
}

var cloneStatusStyles = map[clone.Status]lipgloss.Style{
	clone.Pending: statusStyles[move.Pending],
	clone.Cloned:  statusStyles[move.Moved],
	clone.Failed:  statusStyles[move.Failed],
	clone.Skipped: statusStyles[move.Skipped],
}

// cloneView is the clone review, progress and summary screen.
func (a app) cloneView() string {
	p := a.clonePlan
	header := headerStyle.Render("Clone into " + p.Dir)
	if a.editingDir {
		header = headerStyle.Render("Clone into ") + a.dirInput.View()
	}
	lines := []string{header}
	for _, it := range p.Items {
		label := it.Status.String()
		if a.mode == modeCloneReview && it.Status == clone.Pending {
			label = "clone"
		}
		line := fmt.Sprintf("  %s %s", cloneStatusStyles[it.Status].Render(fmt.Sprintf("%-8s", label)), it.FullName())
		if it.Reason != "" {
			line += "  " + helpStyle.Render(it.Reason)
		}
		if it.Err != nil {
			line += "  " + helpStyle.Render(it.Err.Error())
		}
		lines = append(lines, line)
	}
	lines = append(lines, "")
	lines = a.fit(lines)

	var footer, help string
	switch a.mode {
	case modeCloneReview:
		pending := p.Count(clone.Pending)
		footer = fmt.Sprintf("%d to clone, %d skipped.", pending, p.Count(clone.Skipped))
		if a.status != "" {
			footer = a.status
		}
		help = "y clone · d change directory · esc back to browser"
		if pending == 0 {
			help = "d change directory · esc back to browser"
		}
		if a.editingDir {
			help = "enter use this directory · esc keep " + p.Dir
		}
	case modeCloning:
		done := p.Count(clone.Cloned) + p.Count(clone.Failed)
		footer = fmt.Sprintf("Cloning %d/%d...", done, done+p.Count(clone.Pending))
		if a.status != "" {
			footer = a.status
		}
		help = "ctrl+c abort"
	case modeCloneDone:
		footer = cloneSummary(p)
		help = "any key to return to the browser"
	}
	return strings.Join(lines, "\n") + "\n" + footer + "\n" + helpStyle.Render(help)
}

func cloneSummary(p clone.Plan) string {
	return fmt.Sprintf("%d cloned, %d failed, %d skipped.", p.Count(clone.Cloned), p.Count(clone.Failed), p.Count(clone.Skipped))
}

func summary(plans []move.Plan) string {
	var moved, failed, skipped int
	for _, p := range plans {
		moved += p.Count(move.Moved)
		failed += p.Count(move.Failed)
		skipped += p.Count(move.Skipped)
	}
	return fmt.Sprintf("%d moved, %d failed, %d skipped.", moved, failed, skipped)
}

// Options come from the command line.
type Options struct {
	CloneDir string // where clones go; empty means the current directory
}

// Run starts the TUI and returns what happened.
func Run(opts Options) (Result, error) {
	a := newApp()
	if opts.CloneDir != "" {
		dir, err := clone.ResolveDir(opts.CloneDir, a.startDir)
		if err != nil {
			return Result{}, err
		}
		a.cloneDir = dir
	}
	final, err := tea.NewProgram(a).Run()
	if err != nil {
		return Result{}, err
	}
	a = final.(app)
	if a.err != nil {
		return Result{}, a.err
	}
	ran := a.mode == modeRunning || a.mode == modeDone
	return Result{Ran: ran, Plans: a.plans}, nil
}
