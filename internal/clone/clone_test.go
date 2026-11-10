package clone

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

func TestNewPlanSkipsExisting(t *testing.T) {
	repos := []github.Repo{{Owner: "me", Name: "a"}, {Owner: "org", Name: "here"}}
	exists := func(path string) bool { return path == "/src/here" }
	p := NewPlan("/src", repos, exists)

	if len(p.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(p.Items))
	}
	if it := p.Items[0]; it.FullName() != "me/a" || it.Status != Pending {
		t.Errorf("item 0 = %s/%s", it.FullName(), it.Status)
	}
	if it := p.Items[1]; it.FullName() != "org/here" || it.Status != Skipped {
		t.Errorf("item 1 = %s/%s", it.FullName(), it.Status)
	}

	p.AddMissing("me", []string{"ghost"})
	if it := p.Items[2]; it.FullName() != "me/ghost" || it.Status != Skipped || it.Reason != "not found in me" {
		t.Errorf("item 2 = %+v", it)
	}
}

func TestResolveDir(t *testing.T) {
	base := t.TempDir()
	home, _ := os.UserHomeDir()
	file := filepath.Join(base, "file")
	os.WriteFile(file, nil, 0o644)

	for _, c := range []struct{ in, want string }{
		{"", base},
		{"  ", base},
		{"sub/dir", filepath.Join(base, "sub/dir")},
		{"../up", filepath.Join(filepath.Dir(base), "up")},
		{"/abs/path/", "/abs/path"},
		{"~", home},
		{"~/src", filepath.Join(home, "src")},
	} {
		got, err := ResolveDir(c.in, base)
		if err != nil || got != c.want {
			t.Errorf("ResolveDir(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := ResolveDir("file", base); err == nil {
		t.Error("a file should be refused")
	}
}

func TestExecuteContinuesOnFailure(t *testing.T) {
	p := Plan{Dir: "/src", Items: []Item{
		{Owner: "me", Name: "one"},
		{Owner: "me", Name: "bad"},
		{Owner: "me", Name: "skipme", Status: Skipped},
		{Owner: "org", Name: "three"},
	}}

	var called []string
	clone := func(fullName, dir string) error {
		called = append(called, fullName+"->"+dir)
		if fullName == "me/bad" {
			return errors.New("boom")
		}
		return nil
	}
	var reported int
	Execute(&p, clone, func(Item) { reported++ })

	want := []string{"me/one->/src/one", "me/bad->/src/bad", "org/three->/src/three"}
	if len(called) != len(want) {
		t.Fatalf("called %v, want %v", called, want)
	}
	for i := range want {
		if called[i] != want[i] {
			t.Errorf("call %d = %s, want %s", i, called[i], want[i])
		}
	}
	if reported != 3 {
		t.Errorf("reported %d, want 3", reported)
	}
	if p.Count(Cloned) != 2 || p.Count(Failed) != 1 || p.Count(Skipped) != 1 {
		t.Errorf("counts: cloned=%d failed=%d skipped=%d", p.Count(Cloned), p.Count(Failed), p.Count(Skipped))
	}
}
