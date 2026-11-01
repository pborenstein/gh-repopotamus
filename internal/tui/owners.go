// Package tui is the interactive front end. It shares selection and move
// logic with the CLI and only handles input and rendering.
package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

// ownerItem adapts github.Owner to list.DefaultItem.
type ownerItem github.Owner

func (o ownerItem) FilterValue() string { return o.Login }
func (o ownerItem) Title() string       { return o.Login }
func (o ownerItem) Description() string {
	if o.IsOrg {
		return "org"
	}
	return "you"
}

// ownersExcept returns owners as list items, leaving out skip.
func ownersExcept(owners []github.Owner, skip string) []list.Item {
	items := make([]list.Item, 0, len(owners))
	for _, o := range owners {
		if o.Login != skip {
			items = append(items, ownerItem(o))
		}
	}
	return items
}

type pickStep int

const (
	stepLoading pickStep = iota
	stepSource
	stepDest
	stepDone
)

// ownersLoadedMsg carries the result of the ListOwners call back into Update.
type ownersLoadedMsg struct {
	owners []github.Owner
	err    error
}

func loadOwners() tea.Msg {
	owners, err := github.ListOwners()
	return ownersLoadedMsg{owners, err}
}

// ownerPicker asks for the source owner, then the destination owner.
type ownerPicker struct {
	step   pickStep
	owners []github.Owner
	list   list.Model
	src    string
	dst    string
	err    error
}

func newOwnerPicker() ownerPicker {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowStatusBar(false)
	return ownerPicker{step: stepLoading, list: l}
}

func (m ownerPicker) Init() tea.Cmd {
	return loadOwners
}

func (m ownerPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ownersLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.owners = msg.owners
		m.step = stepSource
		return m, m.showOwners("Move repos FROM", "")

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		// While the filter box is open, enter and esc belong to the list.
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.choose()
		case "esc":
			if m.step == stepDest && m.list.FilterState() == list.Unfiltered {
				m.step = stepSource
				m.src = ""
				return m, m.showOwners("Move repos FROM", "")
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// showOwners refills the list for a new step, with a clear filter and the
// cursor back on the first row.
func (m *ownerPicker) showOwners(title, skip string) tea.Cmd {
	m.list.Title = title
	m.list.ResetFilter()
	cmd := m.list.SetItems(ownersExcept(m.owners, skip))
	m.list.Select(0)
	return cmd
}

// choose records the highlighted owner and advances to the next step.
func (m ownerPicker) choose() (tea.Model, tea.Cmd) {
	item, ok := m.list.SelectedItem().(ownerItem)
	if !ok {
		return m, nil
	}
	switch m.step {
	case stepSource:
		m.src = item.Login
		m.step = stepDest
		return m, m.showOwners("Move repos from "+m.src+" TO", m.src)
	case stepDest:
		m.dst = item.Login
		m.step = stepDone
		return m, tea.Quit
	}
	return m, nil
}

func (m ownerPicker) View() tea.View {
	content := m.list.View()
	if m.step == stepLoading {
		content = "Loading owners..."
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// PickOwners runs the owner picker. ok is false if the user quit without
// choosing both owners.
func PickOwners() (src, dst string, ok bool, err error) {
	final, err := tea.NewProgram(newOwnerPicker()).Run()
	if err != nil {
		return "", "", false, err
	}
	m := final.(ownerPicker)
	if m.err != nil {
		return "", "", false, m.err
	}
	return m.src, m.dst, m.step == stepDone, nil
}
