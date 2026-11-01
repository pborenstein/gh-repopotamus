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

func press(m tea.Model, key string) tea.Model {
	var k tea.KeyPressMsg
	switch key {
	case "enter":
		k = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		k = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "down":
		k = tea.KeyPressMsg{Code: tea.KeyDown}
	}
	m, _ = m.Update(k)
	return m
}

func TestPickFlow(t *testing.T) {
	var m tea.Model = newOwnerPicker()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ownersLoadedMsg{owners: testOwners})

	// Source: first item, "me".
	m = press(m, "enter")
	p := m.(ownerPicker)
	if p.step != stepDest || p.src != "me" {
		t.Fatalf("after source pick: step=%v src=%q", p.step, p.src)
	}

	// Esc goes back to the source step.
	m = press(m, "esc")
	if p := m.(ownerPicker); p.step != stepSource || p.src != "" {
		t.Fatalf("after esc: step=%v src=%q", p.step, p.src)
	}

	// Pick "org-a" as source, then the first remaining owner ("me") as dest.
	m = press(m, "down")
	m = press(m, "enter")
	m = press(m, "enter")
	p = m.(ownerPicker)
	if p.step != stepDone || p.src != "org-a" || p.dst != "me" {
		t.Fatalf("final: step=%v src=%q dst=%q", p.step, p.src, p.dst)
	}
}
