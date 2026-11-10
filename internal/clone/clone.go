// Package clone turns a list of repos into a clone plan and carries it out.
// Each repo is cloned into its own folder, named after the repo, under one
// directory.
package clone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

type Status int

const (
	Pending Status = iota
	Cloned
	Failed
	Skipped
)

func (s Status) String() string {
	return [...]string{"pending", "cloned", "failed", "skipped"}[s]
}

// Item is one repo in a plan. Reason explains a skip; Err holds a failure.
type Item struct {
	Owner  string
	Name   string
	Status Status
	Reason string
	Err    error
}

func (it Item) FullName() string {
	return it.Owner + "/" + it.Name
}

type Plan struct {
	Dir   string
	Items []Item
}

// Path is where it gets cloned to.
func (p Plan) Path(it Item) string {
	return filepath.Join(p.Dir, it.Name)
}

// Exists reports whether anything is at path. NewPlan uses it to skip repos
// that would clone over an existing file or folder.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ResolveDir turns what the user typed into an absolute directory. Empty
// means base; a leading ~ is the home directory; a relative path is taken
// from base. The directory may not exist yet (gh repo clone creates it),
// but it can't be a file.
func ResolveDir(input, base string) (string, error) {
	dir := strings.TrimSpace(input)
	switch {
	case dir == "":
		dir = base
	case dir == "~" || strings.HasPrefix(dir, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, dir[1:])
	case !filepath.IsAbs(dir):
		dir = filepath.Join(base, dir)
	}
	dir = filepath.Clean(dir)
	if fi, err := os.Stat(dir); err == nil && !fi.IsDir() {
		return "", fmt.Errorf("%s is a file, not a directory", dir)
	}
	return dir, nil
}

// NewPlan builds a plan to clone repos into dir. A repo whose folder already
// exists is skipped, because gh repo clone would fail on it.
func NewPlan(dir string, repos []github.Repo, exists func(string) bool) Plan {
	p := Plan{Dir: dir}
	for _, r := range repos {
		it := Item{Owner: r.Owner, Name: r.Name}
		if exists(p.Path(it)) {
			it.Status = Skipped
			it.Reason = "already exists"
		}
		p.Items = append(p.Items, it)
	}
	return p
}

// AddMissing records names that weren't found in owner as skipped, so the
// plan shows them alongside the rest.
func (p *Plan) AddMissing(owner string, names []string) {
	for _, n := range names {
		p.Items = append(p.Items, Item{Owner: owner, Name: n, Status: Skipped, Reason: fmt.Sprintf("not found in %s", owner)})
	}
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

// CloneFunc matches github.Clone; tests pass a fake.
type CloneFunc func(fullName, dir string) error

// Execute clones each pending item in order and keeps going after a failure.
// progress, if set, is called after each attempt.
func Execute(p *Plan, clone CloneFunc, progress func(Item)) {
	for i := range p.Items {
		it := &p.Items[i]
		if it.Status != Pending {
			continue
		}
		if err := clone(it.FullName(), p.Path(*it)); err != nil {
			it.Status = Failed
			it.Err = err
		} else {
			it.Status = Cloned
		}
		if progress != nil {
			progress(*it)
		}
	}
}
