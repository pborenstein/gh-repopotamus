// Package move turns a selection into a transfer plan and carries it out.
// The CLI and the TUI share it.
package move

import (
	"fmt"
	"strings"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

type Status int

const (
	Pending Status = iota
	Moved
	Failed
	Skipped
)

func (s Status) String() string {
	return [...]string{"pending", "moved", "failed", "skipped"}[s]
}

// Item is one repo in a plan. Reason explains a skip; Err holds a failure.
type Item struct {
	Name   string
	Status Status
	Reason string
	Err    error
}

type Plan struct {
	Src   string
	Dst   string
	Items []Item
}

// NewPlan builds a plan. Names in missing weren't found in src. A selected
// repo whose name already exists in dst is skipped, because the transfer
// would fail with 422.
func NewPlan(src, dst string, selected []github.Repo, missing []string, dstRepos []github.Repo) Plan {
	taken := make(map[string]bool, len(dstRepos))
	for _, r := range dstRepos {
		taken[strings.ToLower(r.Name)] = true
	}

	p := Plan{Src: src, Dst: dst}
	for _, r := range selected {
		it := Item{Name: r.Name}
		if taken[strings.ToLower(r.Name)] {
			it.Status = Skipped
			it.Reason = fmt.Sprintf("%s already has a repo with this name", dst)
		}
		p.Items = append(p.Items, it)
	}
	for _, n := range missing {
		p.Items = append(p.Items, Item{Name: n, Status: Skipped, Reason: fmt.Sprintf("not found in %s", src)})
	}
	return p
}

// Count returns how many items have status s.
func (p Plan) Count(s Status) int {
	n := 0
	for _, it := range p.Items {
		if it.Status == s {
			n++
		}
	}
	return n
}

// TransferFunc matches github.Transfer; tests pass a fake.
type TransferFunc func(owner, repo, newOwner string) error

// Execute transfers each pending item in order and keeps going after a
// failure. progress, if set, is called after each attempt.
func Execute(p *Plan, transfer TransferFunc, progress func(Item)) {
	for i := range p.Items {
		it := &p.Items[i]
		if it.Status != Pending {
			continue
		}
		if err := transfer(p.Src, it.Name, p.Dst); err != nil {
			it.Status = Failed
			it.Err = err
		} else {
			it.Status = Moved
		}
		if progress != nil {
			progress(*it)
		}
	}
}
