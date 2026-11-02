package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

var testOwners = []github.Owner{
	{Login: "me"},
	{Login: "org-a", IsOrg: true},
	{Login: "org-b", IsOrg: true},
}

// keyPress builds a key press for the names used in these tests.
func keyPress(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(k)[0]
	return tea.KeyPressMsg{Code: r, Text: k}
}

func TestOwnersExcept(t *testing.T) {
	items := ownersExcept(testOwners, "org-a")
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	for _, it := range items {
		if it.(ownerItem).Login == "org-a" {
			t.Errorf("org-a should be excluded")
		}
	}
}

func TestOwnerPicker(t *testing.T) {
	m := newOwnerPicker()
	m.list.SetSize(40, 20)
	m.show(testOwners, "me", "org-b")

	if got := m.list.SelectedItem().(ownerItem).Login; got != "org-b" {
		t.Fatalf("cursor on %q, want current owner org-b", got)
	}
	_, cmd := m.Update(keyPress("enter"))
	if got, ok := cmd().(ownerChosenMsg); !ok || got.login != "org-b" {
		t.Fatalf("enter sent %#v", cmd())
	}
	_, cmd = m.Update(keyPress("esc"))
	if _, ok := cmd().(ownerCancelMsg); !ok {
		t.Fatalf("esc sent %#v", cmd())
	}
}
