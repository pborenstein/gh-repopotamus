package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pborenstein/gh-repopotamus/internal/clone"
	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/move"
)

// harness drives an app the way Bubble Tea would, running each returned
// command and feeding its message back in.
type harness struct {
	t         *testing.T
	a         app
	transfers []string
	clones    []string
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
	h.a.startDir, h.a.cloneDir = "/src", "/src"
	h.a.exists = func(path string) bool { return path == "/src/Shared" }
	h.a.cloneRepo = func(fullName, dir string) error {
		h.clones = append(h.clones, fullName+"->"+dir)
		if fullName == "me/beta" {
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
	case reposLoadedMsg, ownerChosenMsg, ownerCancelMsg, progressMsg, transfersDoneMsg,
		cloneProgressMsg, clonesDoneMsg:
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

// With nothing selected, c clones every repo the pane shows. A folder that
// already exists is skipped, and finishing returns to the browser.
func TestCloneAllShown(t *testing.T) {
	h := newHarness(t)
	h.keys("enter", "tab", "c")
	if h.a.mode != modeCloneReview {
		t.Fatalf("after c: mode=%v status=%q", h.a.mode, h.a.status)
	}
	p := h.a.clonePlan
	if p.Count(clone.Pending) != 2 || p.Count(clone.Skipped) != 1 {
		t.Fatalf("plan = %+v", p.Items)
	}

	h.keys("y")
	if h.a.mode != modeCloneDone {
		t.Fatalf("after y: mode=%v", h.a.mode)
	}
	if got := strings.Join(h.clones, " "); got != "me/alpha->/src/alpha me/beta->/src/beta" {
		t.Errorf("clones = %s", got)
	}

	h.keys("x")
	if h.a.mode != modeBrowse || h.a.status != "1 cloned, 1 failed, 1 skipped." {
		t.Errorf("after done: mode=%v status=%q", h.a.mode, h.a.status)
	}
}

// With a selection, c clones only the selection. A repo queued into the
// pane is cloned from the owner it lives in today.
func TestCloneSelected(t *testing.T) {
	h := newHarness(t)
	h.keys("enter", "tab", ">", "tab") // queue alpha to org-a, go to org-a
	h.keys("space", "c")               // select +alpha (first row)
	if h.a.mode != modeCloneReview {
		t.Fatalf("after c: mode=%v status=%q", h.a.mode, h.a.status)
	}
	items := h.a.clonePlan.Items
	if len(items) != 1 || items[0].FullName() != "me/alpha" {
		t.Errorf("plan = %+v", items)
	}
}

func TestCloneReviewEscReturns(t *testing.T) {
	h := newHarness(t)
	h.keys("enter", "tab", "c", "esc")
	if h.a.mode != modeBrowse || len(h.clones) != 0 {
		t.Fatalf("mode=%v clones=%v", h.a.mode, h.clones)
	}
}

// d edits the clone directory. A relative path starts from where the app
// was started, and the plan is rebuilt, so skips match the new directory.
func TestCloneChangeDir(t *testing.T) {
	h := newHarness(t)
	h.keys("enter", "tab", "c", "d")
	if !h.a.editingDir || h.a.dirInput.Value() != "/src" {
		t.Fatalf("editing=%v value=%q", h.a.editingDir, h.a.dirInput.Value())
	}
	h.a.dirInput.SetValue("sub")
	h.keys("2", "enter")
	if h.a.editingDir || h.a.cloneDir != "/src/sub2" {
		t.Fatalf("editing=%v dir=%q status=%q", h.a.editingDir, h.a.cloneDir, h.a.status)
	}
	// /src/Shared existed; /src/sub2/Shared doesn't.
	if p := h.a.clonePlan; p.Dir != "/src/sub2" || p.Count(clone.Pending) != 3 {
		t.Errorf("plan = %s %+v", p.Dir, p.Items)
	}

	// A file is refused and editing continues; esc keeps the old directory.
	h.keys("d")
	h.a.dirInput.SetValue("/etc/hosts")
	h.keys("enter")
	if !h.a.editingDir || !strings.Contains(h.a.status, "not a directory") {
		t.Errorf("editing=%v status=%q", h.a.editingDir, h.a.status)
	}
	h.keys("esc")
	if h.a.editingDir || h.a.cloneDir != "/src/sub2" || h.a.mode != modeCloneReview {
		t.Errorf("after esc: editing=%v dir=%q mode=%v", h.a.editingDir, h.a.cloneDir, h.a.mode)
	}

	// The choice holds for the next clone.
	h.keys("esc", "c")
	if h.a.clonePlan.Dir != "/src/sub2" {
		t.Errorf("next plan dir = %q", h.a.clonePlan.Dir)
	}
}
