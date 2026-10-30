package move

import (
	"errors"
	"testing"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

func TestNewPlan(t *testing.T) {
	selected := []github.Repo{{Name: "a"}, {Name: "Clash"}}
	dst := []github.Repo{{Name: "clash"}, {Name: "other"}}
	p := NewPlan("src", "dst", selected, []string{"ghost"}, dst)

	want := []struct {
		name   string
		status Status
	}{{"a", Pending}, {"Clash", Skipped}, {"ghost", Skipped}}
	if len(p.Items) != len(want) {
		t.Fatalf("got %d items, want %d", len(p.Items), len(want))
	}
	for i, w := range want {
		if p.Items[i].Name != w.name || p.Items[i].Status != w.status {
			t.Errorf("item %d = %s/%s, want %s/%s", i, p.Items[i].Name, p.Items[i].Status, w.name, w.status)
		}
	}
}

func TestExecuteContinuesOnFailure(t *testing.T) {
	p := Plan{Src: "src", Dst: "dst", Items: []Item{
		{Name: "one"},
		{Name: "bad"},
		{Name: "skipme", Status: Skipped},
		{Name: "three"},
	}}

	var called []string
	transfer := func(owner, repo, newOwner string) error {
		if owner != "src" || newOwner != "dst" {
			t.Errorf("transfer(%s, %s, %s): wrong owners", owner, repo, newOwner)
		}
		called = append(called, repo)
		if repo == "bad" {
			return errors.New("boom")
		}
		return nil
	}
	var reported int
	Execute(&p, transfer, func(Item) { reported++ })

	if len(called) != 3 || called[2] != "three" {
		t.Errorf("called = %v, want one, bad, three", called)
	}
	if reported != 3 {
		t.Errorf("progress called %d times, want 3", reported)
	}
	if p.Count(Moved) != 2 || p.Count(Failed) != 1 || p.Count(Skipped) != 1 {
		t.Errorf("moved/failed/skipped = %d/%d/%d, want 2/1/1", p.Count(Moved), p.Count(Failed), p.Count(Skipped))
	}
	if p.Items[1].Err == nil {
		t.Error("failed item should keep its error")
	}
}
