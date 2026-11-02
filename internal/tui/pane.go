package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	accent        = lipgloss.Color("62")
	incomingColor = lipgloss.Color("214")
	cursorStyle   = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
	incomingStyle = lipgloss.NewStyle().Foreground(incomingColor)
)

// rowItem adapts row to list.Item. Filtering matches the repo name.
type rowItem struct{ row }

func (r rowItem) FilterValue() string { return r.repo.Name }

// rowDelegate renders one repo per line. It shares the pane's selected map
// (maps are references), so checkboxes stay current without copying.
type rowDelegate struct {
	selected map[string]bool
	nameMax  int // longest name in the pane
	width    int // list width
}

func (d rowDelegate) Height() int                         { return 1 }
func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	r, ok := item.(rowItem)
	if !ok {
		return
	}
	fmt.Fprint(w, formatRow(r.row, d.selected[r.key()], index == m.Index(), d.nameMax, d.width))
}

// column is one optional field after the name.
type column struct {
	width int
	value func(row) string
}

// columns in the order they're dropped when the pane is narrow: last first.
var columns = []column{
	{7, func(r row) string { return strings.ToLower(r.repo.Visibility) }},
	{9, func(r row) string {
		var f []string
		if r.repo.Archived {
			f = append(f, "arch")
		}
		if r.repo.Fork {
			f = append(f, "fork")
		}
		return strings.Join(f, ",")
	}},
	{7, func(r row) string {
		if r.repo.PushedAt.IsZero() {
			return "-"
		}
		return r.repo.PushedAt.Format("2006-01")
	}},
	{10, func(r row) string { return r.repo.Language }},
}

const minName = 16

// layout picks how wide the name is and how many columns fit in width.
func layout(nameMax, width int) (nameW, ncols int) {
	avail := width - 7 // cursor (2) + box (3) + space (1) + incoming mark (1)
	ncols = len(columns)
	for ncols > 0 {
		used := 0
		for _, c := range columns[:ncols] {
			used += c.width + 1
		}
		if avail-used >= min(nameMax, minName) {
			return min(nameMax, avail-used), ncols
		}
		ncols--
	}
	return max(avail, 1), 0
}

// formatRow renders: cursor, checkbox, incoming mark, name, then as many
// columns as fit.
func formatRow(r row, selected, current bool, nameMax, width int) string {
	nameW, ncols := layout(nameMax, width)
	cursor, box, mark := "  ", "[ ]", " "
	if current {
		cursor = "> "
	}
	if selected {
		box = "[x]"
	}
	if r.incoming {
		mark = "+"
	}
	name := r.repo.Name
	if len(name) > nameW {
		name = name[:max(nameW-1, 0)] + "…"
	}
	name = fmt.Sprintf("%-*s", nameW, name)

	var meta []string
	for _, c := range columns[:ncols] {
		v := c.value(r)
		if len(v) > c.width {
			v = v[:c.width]
		}
		meta = append(meta, fmt.Sprintf("%-*s", c.width, v))
	}
	rest := " " + strings.Join(meta, " ")
	if ncols == 0 {
		rest = ""
	}

	head := cursor + box + mark + name
	switch {
	case r.incoming && current:
		return cursorStyle.Foreground(incomingColor).Render(head) + rest
	case r.incoming:
		return incomingStyle.Render(head) + dimStyle.Render(rest)
	case current:
		return cursorStyle.Render(head) + rest
	}
	return head + dimStyle.Render(rest)
}

// pane is one side of the browser: an owner's projected repo list.
type pane struct {
	owner        string
	loading      bool
	rows         []row
	selected     map[string]bool
	hideArchived bool
	hideForks    bool
	list         list.Model
	width        int
}

func newPane() pane {
	selected := map[string]bool{}
	l := newList(rowDelegate{selected: selected})
	l.KeyMap.Quit.SetEnabled(false)
	l.SetShowHelp(false)
	l.SetStatusBarItemName("repo", "repos")
	l.Styles.Title = l.Styles.Title.Background(accent)
	return pane{selected: selected, list: l}
}

// open points the pane at a new owner. Selection and toggles reset.
func (p *pane) open(owner string) {
	p.owner = owner
	p.loading = true
	p.rows = nil
	clear(p.selected)
	p.hideArchived, p.hideForks = false, false
	p.list.ResetFilter()
	p.list.SetItems(nil)
	p.setTitle()
}

func (p *pane) setSize(width, height int) {
	p.width = width
	// bubbles' list renders one line more than its height once it has more
	// than one page (a blank line above the pagination dots isn't counted).
	// Give it one line less so the pane never overflows.
	p.list.SetSize(width, max(height-1, 1))
	p.setDelegate()
}

func (p *pane) setDelegate() {
	nameMax := 0
	for _, r := range p.rows {
		nameMax = max(nameMax, len(r.repo.Name))
	}
	p.list.SetDelegate(rowDelegate{selected: p.selected, nameMax: nameMax, width: p.width})
}

// setRows replaces the pane's rows and reapplies the toggles. Selected rows
// that are gone or hidden are unselected, so nothing hidden gets moved.
func (p *pane) setRows(rows []row) tea.Cmd {
	p.loading = false
	p.rows = rows
	p.setDelegate()
	keep := map[string]bool{}
	var items []list.Item
	for _, r := range rows {
		if (p.hideArchived && r.repo.Archived) || (p.hideForks && r.repo.Fork) {
			continue
		}
		keep[r.key()] = true
		items = append(items, rowItem{r})
	}
	for k := range p.selected {
		if !keep[k] {
			delete(p.selected, k)
		}
	}
	p.setTitle()
	cmd := p.list.SetItems(items)
	// SetItems leaves the cursor where it was, which can be past the end
	// after rows move out.
	if n := len(p.list.VisibleItems()); n > 0 && p.list.Index() >= n {
		p.list.Select(n - 1)
	}
	return cmd
}

func (p *pane) setTitle() {
	if p.owner == "" {
		p.list.Title = "(no owner)"
		return
	}
	t := p.owner
	if n := len(p.selected); n > 0 {
		t += fmt.Sprintf(" · %d selected", n)
	}
	if p.hideArchived {
		t += " · -archived"
	}
	if p.hideForks {
		t += " · -forks"
	}
	p.list.Title = t
}

// set marks or unmarks a row. Unselected rows are deleted from the map so
// len(selected) is the count.
func (p *pane) set(key string, on bool) {
	if on {
		p.selected[key] = true
	} else {
		delete(p.selected, key)
	}
}

func (p *pane) toggleCurrent() {
	if r, ok := p.list.SelectedItem().(rowItem); ok {
		p.set(r.key(), !p.selected[r.key()])
		p.setTitle()
	}
}

// selectAllShown selects every row the filter shows, or clears them if they
// are all selected already.
func (p *pane) selectAllShown() {
	shown := p.list.VisibleItems()
	allOn := true
	for _, it := range shown {
		if !p.selected[it.(rowItem).key()] {
			allOn = false
			break
		}
	}
	for _, it := range shown {
		p.set(it.(rowItem).key(), !allOn)
	}
	p.setTitle()
}

func (p *pane) invertShown() {
	for _, it := range p.list.VisibleItems() {
		k := it.(rowItem).key()
		p.set(k, !p.selected[k])
	}
	p.setTitle()
}

// chosen returns the selected rows, or the row under the cursor if nothing
// is selected (the way DA Mover worked on the highlighted item).
func (p pane) chosen() []row {
	var out []row
	for _, r := range p.rows {
		if p.selected[r.key()] {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		if r, ok := p.list.SelectedItem().(rowItem); ok {
			out = append(out, r.row)
		}
	}
	return out
}

func (p pane) filtering() bool {
	return p.list.FilterState() == list.Filtering
}

func (p pane) View() string {
	switch {
	case p.owner == "":
		return "Press o to open an owner."
	case p.loading:
		return fmt.Sprintf("Loading %s...", p.owner)
	}
	return p.list.View()
}
