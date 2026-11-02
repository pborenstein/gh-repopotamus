package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/move"
)

// harness drives an app the way Bubble Tea would, running each returned
// command and feeding its message back in.
type harness struct {
	t         *testing.T
	a         app
	transfers []string
}

func newHarness(t *testing.T) *harness {
	h := &harness{t: t, a: newApp()}
	h.a.listRepos = func(owner string) ([]github.Repo, error) {
		return testCache[owner], nil
	}
	h.a.transfer = func(owner, name, to string) error {
		h.transfers = append(h.transfers, owner+"/"+name+"->"+to)
		if name == "beta" {
			return errors.New("boom")
		}
		return nil
	}
	h.send(tea.WindowSizeMsg{Width: 120, Height: 30})
	h.send(ownersLoadedMsg{owners: testOwners})
	return h
}

// send delivers msg and then runs the resulting commands to completion.
// Commands that produce nothing useful here (ticks, nil) are dropped.
func (h *harness) send(msg tea.Msg) {
	queue := []tea.Msg{msg}
	for len(queue) > 0 && len(queue) < 1000 {
		msg, queue = queue[0], queue[1:]
		m, cmd := h.a.Update(msg)
		h.a = m.(app)
		queue = append(queue, run(cmd)...)
	}
}

// run executes cmd, expanding batches, and keeps only messages the app
// cares about in these tests.
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range msg {
			out = append(out, run(c)...)
		}
		return out
	case reposLoadedMsg, ownerChosenMsg, ownerCancelMsg, progressMsg, transfersDoneMsg:
		return []tea.Msg{msg}
	}
	return nil
}

func (h *harness) keys(ks ...string) {
	for _, k := range ks {
		h.send(keyPress(k))
	}
}

func TestBrowseQueueAndRun(t *testing.T) {
	h := newHarness(t)
	a := func() app { return h.a }

	// Left pane opens on "me"; the owner picker is open on the right.
	if a().mode != modePickOwner || a().active != 1 || a().panes[0].owner != "me" {
		t.Fatalf("start: mode=%v active=%d left=%q", a().mode, a().active, a().panes[0].owner)
	}
	if n := len(a().panes[0].rows); n != 3 {
		t.Fatalf("left pane rows = %d, want 3", n)
	}

	h.keys("enter") // open org-a (first owner after skipping "me")
	if a().mode != modeBrowse || a().panes[1].owner != "org-a" {
		t.Fatalf("after pick: mode=%v right=%q", a().mode, a().panes[1].owner)
	}

	// Left pane: select alpha and beta, move right.
	h.keys("tab", "space", "down", "space", ">")
	if len(a().queue) != 2 {
		t.Fatalf("queue = %+v, status %q", a().queue, a().status)
	}
	if got := rowNames(a().panes[1].rows); strings.Join(got, ",") != "+alpha,+beta,handbook,shared" {
		t.Errorf("right pane = %v", got)
	}

	// With nothing selected, > moves the row under the cursor (Shared), which
	// collides with org-a's "shared".
	h.keys(">")
	if !strings.Contains(a().status, "skipped Shared") {
		t.Errorf("status = %q", a().status)
	}

	// q opens the review; y runs it.
	h.keys("q")
	if a().mode != modeReview || countPending(a().plans) != 2 {
		t.Fatalf("review: mode=%v pending=%d", a().mode, countPending(a().plans))
	}
	h.keys("y")
	if a().mode != modeDone {
		t.Fatalf("after y: mode=%v", a().mode)
	}
	if got := strings.Join(h.transfers, " "); got != "me/alpha->org-a me/beta->org-a" {
		t.Errorf("transfers = %s", got)
	}
	p := a().plans[0]
	if p.Count(move.Moved) != 1 || p.Count(move.Failed) != 1 {
		t.Errorf("plan after run = %+v", p)
	}
}

func TestQuitWithEmptyQueue(t *testing.T) {
	h := newHarness(t)
	h.keys("enter")
	_, cmd := h.a.Update(keyPress("q"))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q with nothing queued should quit")
	}
}

func TestReviewEscReturns(t *testing.T) {
	h := newHarness(t)
	h.keys("enter", "tab", "space", ">", "q", "esc")
	if h.a.mode != modeBrowse || len(h.a.queue) != 1 {
		t.Fatalf("mode=%v queue=%d", h.a.mode, len(h.a.queue))
	}
}

func TestLayoutDropsColumns(t *testing.T) {
	_, wide := layout(20, 120)
	_, narrow := layout(20, 40)
	if wide != len(columns) || narrow >= wide {
		t.Errorf("wide=%d narrow=%d", wide, narrow)
	}
	nameW, _ := layout(80, 60)
	if nameW < minName {
		t.Errorf("nameW = %d, want >= %d", nameW, minName)
	}
}

// A frame taller than the terminal leaves stale lines on screen, so the
// browser must fit exactly, including when a pane has several pages.
func TestBrowserFitsScreen(t *testing.T) {
	h := newHarness(t)
	var many []github.Repo
	for i := 0; i < 72; i++ {
		many = append(many, repo("me", fmt.Sprintf("repo-%02d", i)))
	}
	h.a.listRepos = func(owner string) ([]github.Repo, error) {
		if owner == "me" {
			return many, nil
		}
		return testCache[owner], nil
	}
	h.a.cache = map[string][]github.Repo{}
	h.send(ownersLoadedMsg{owners: testOwners})
	h.keys("enter")
	for _, height := range []int{12, 30, 51} {
		h.send(tea.WindowSizeMsg{Width: 120, Height: height})
		if got := strings.Count(h.a.View().Content, "\n") + 1; got != height {
			t.Errorf("terminal height %d: view is %d lines", height, got)
		}
	}
}
