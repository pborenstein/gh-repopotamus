package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pborenstein/gh-repopotamus/internal/github"
	"github.com/pborenstein/gh-repopotamus/internal/move"
)

// staged is one queued move. from is the repo's real owner today, so
// re-routing a queued repo never chains transfers.
type staged struct {
	repo     github.Repo
	from, to string
}

// queue holds moves the user has staged but not confirmed. Panes render the
// projected state: what each owner will have once the queue runs.
type queue []staged

// row is one line in a pane. origin is the repo's real owner. incoming
// means it is queued to move into the pane's owner.
type row struct {
	repo     github.Repo
	origin   string
	incoming bool
}

func (r row) key() string { return r.origin + "/" + r.repo.Name }

// project returns owner's rows once the queue runs: its real repos minus the
// ones queued away, plus the ones queued in. Sorted by name.
func (q queue) project(owner string, actual []github.Repo) []row {
	away := map[string]bool{}
	for _, s := range q {
		if s.from == owner {
			away[strings.ToLower(s.repo.Name)] = true
		}
	}
	var rows []row
	for _, r := range actual {
		if !away[strings.ToLower(r.Name)] {
			rows = append(rows, row{repo: r, origin: owner})
		}
	}
	for _, s := range q {
		if s.to == owner {
			rows = append(rows, row{repo: s.repo, origin: s.from, incoming: true})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].repo.Name) < strings.ToLower(rows[j].repo.Name)
	})
	return rows
}

// stage queues r to move to owner to. destRows is to's projected state, used
// to refuse name collisions. Moving a queued repo back to its origin cancels
// it; moving it to a third owner re-routes it.
func (q queue) stage(r row, to string, destRows []row) (queue, error) {
	for _, d := range destRows {
		if strings.EqualFold(d.repo.Name, r.repo.Name) {
			return q, fmt.Errorf("%s already has %s", to, d.repo.Name)
		}
	}
	for i, s := range q {
		if s.from == r.origin && s.repo.Name == r.repo.Name {
			if to == r.origin {
				return append(q[:i:i], q[i+1:]...), nil
			}
			out := append(queue{}, q...)
			out[i].to = to
			return out, nil
		}
	}
	if to == r.origin {
		return q, nil
	}
	return append(q[:len(q):len(q)], staged{repo: r.repo, from: r.origin, to: to}), nil
}

// plans groups the queue into one move.Plan per from/to pair, in the order
// the pairs were first queued. cache supplies each destination's real repos
// so move.NewPlan can still skip collisions.
func (q queue) plans(cache map[string][]github.Repo) []move.Plan {
	var out []move.Plan
	index := map[[2]string]int{}
	selected := map[[2]string][]github.Repo{}
	for _, s := range q {
		k := [2]string{s.from, s.to}
		if _, ok := index[k]; !ok {
			index[k] = len(out)
			out = append(out, move.Plan{Src: s.from, Dst: s.to})
		}
		selected[k] = append(selected[k], s.repo)
	}
	for i, p := range out {
		k := [2]string{p.Src, p.Dst}
		out[i] = move.NewPlan(p.Src, p.Dst, selected[k], nil, cache[p.Dst])
	}
	return out
}
