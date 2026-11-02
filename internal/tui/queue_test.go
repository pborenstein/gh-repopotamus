package tui

import (
	"reflect"
	"testing"

	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/move"
)

func repo(owner, name string) github.Repo { return github.Repo{Owner: owner, Name: name} }

var testCache = map[string][]github.Repo{
	"me":    {repo("me", "alpha"), repo("me", "beta"), repo("me", "Shared")},
	"org-a": {repo("org-a", "handbook"), repo("org-a", "shared")},
	"org-b": {},
}

func rowNames(rows []row) []string {
	var out []string
	for _, r := range rows {
		name := r.repo.Name
		if r.incoming {
			name = "+" + name
		}
		out = append(out, name)
	}
	return out
}

func find(rows []row, name string) row {
	for _, r := range rows {
		if r.repo.Name == name {
			return r
		}
	}
	panic("no row " + name)
}

func TestQueue(t *testing.T) {
	var q queue
	project := func(owner string) []row { return q.project(owner, testCache[owner]) }
	stage := func(name, from, to string) error {
		var err error
		q, err = q.stage(find(project(from), name), to, project(to))
		return err
	}
	expect := func(owner string, want ...string) {
		t.Helper()
		if got := rowNames(project(owner)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", owner, got, want)
		}
	}

	if err := stage("alpha", "me", "org-a"); err != nil {
		t.Fatal(err)
	}
	expect("me", "beta", "Shared")
	expect("org-a", "+alpha", "handbook", "shared")

	// Collision with an existing repo is refused, case-insensitively.
	if err := stage("Shared", "me", "org-a"); err == nil {
		t.Error("staging Shared into org-a should collide with shared")
	}

	// Re-route alpha from org-a on to org-b: still one entry, from me.
	if err := stage("alpha", "org-a", "org-b"); err != nil {
		t.Fatal(err)
	}
	if len(q) != 1 || q[0].from != "me" || q[0].to != "org-b" {
		t.Fatalf("after re-route: %+v", q)
	}
	expect("org-a", "handbook", "shared")
	expect("org-b", "+alpha")

	// Moving it back home cancels it.
	if err := stage("alpha", "org-b", "me"); err != nil {
		t.Fatal(err)
	}
	if len(q) != 0 {
		t.Fatalf("after return: %+v", q)
	}
	expect("me", "alpha", "beta", "Shared")
}

func TestQueuePlans(t *testing.T) {
	q := queue{
		{repo: repo("me", "alpha"), from: "me", to: "org-a"},
		{repo: repo("org-a", "handbook"), from: "org-a", to: "me"},
		{repo: repo("me", "beta"), from: "me", to: "org-a"},
	}
	plans := q.plans(testCache)
	if len(plans) != 2 {
		t.Fatalf("got %d plans, want 2", len(plans))
	}
	if p := plans[0]; p.Src != "me" || p.Dst != "org-a" || len(p.Items) != 2 || p.Count(move.Pending) != 2 {
		t.Errorf("plan 0 = %+v", p)
	}
	if p := plans[1]; p.Src != "org-a" || p.Dst != "me" || p.Items[0].Name != "handbook" {
		t.Errorf("plan 1 = %+v", p)
	}
}
