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

// ownersLoadedMsg carries the result of the ListOwners call back into Update.
type ownersLoadedMsg struct {
	owners []github.Owner
}

// ownerChosenMsg and ownerCancelMsg end the owner picker.
type ownerChosenMsg struct{ login string }
type ownerCancelMsg struct{}

func loadOwners() tea.Msg {
	owners, err := github.ListOwners()
	if err != nil {
		return errMsg{err}
	}
	return ownersLoadedMsg{owners}
}

// ownerPicker is the list that opens inside a pane when you press o.
type ownerPicker struct {
	list list.Model
}

func newOwnerPicker() ownerPicker {
	l := newList(list.NewDefaultDelegate())
	l.KeyMap.Quit.SetEnabled(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.Title = "Open owner"
	return ownerPicker{list: l}
}

// show refills the picker, leaving out the owner open in the other pane, and
// puts the cursor on current if it's listed.
func (m *ownerPicker) show(owners []github.Owner, skip, current string) {
	m.list.ResetFilter()
	items := ownersExcept(owners, skip)
	m.list.SetItems(items)
	m.list.Select(0)
	for i, it := range items {
		if it.(ownerItem).Login == current {
			m.list.Select(i)
		}
	}
}

func (m ownerPicker) Update(msg tea.Msg) (ownerPicker, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && m.list.FilterState() != list.Filtering {
		switch k.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(ownerItem); ok {
				return m, send(ownerChosenMsg{item.Login})
			}
			return m, nil
		case "esc":
			if m.list.FilterState() == list.Unfiltered {
				return m, send(ownerCancelMsg{})
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ownerPicker) View() string {
	return m.list.View()
}
