// Package selection narrows a list of repos down to the ones a command should
// act on. Everything here is pure so it can be tested without GitHub.
package selection

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

// Tri is a filter that can be unset, required true, or required false.
type Tri int

const (
	Any Tri = iota
	Only
	Exclude
)

func (t Tri) allows(v bool) bool {
	switch t {
	case Only:
		return v
	case Exclude:
		return !v
	}
	return true
}

// Criteria describes a selection. Names, if any, form the base set; otherwise
// the base set is every repo. The remaining fields narrow the base set.
type Criteria struct {
	Names        []string
	Match        string // glob, matched against the repo name
	Regex        *regexp.Regexp
	Topics       []string // repo must have all of them
	Visibility   string   // public, private, internal; empty means any
	Language     string   // case-insensitive; empty means any
	PushedBefore time.Time
	Archived     Tri
	Forks        Tri
}

// Narrows reports whether any filter is set. A selection with no names and no
// filters means "every repo", which callers may want to guard.
func (c Criteria) Narrows() bool {
	return c.Match != "" || c.Regex != nil || len(c.Topics) > 0 ||
		c.Visibility != "" || c.Language != "" || !c.PushedBefore.IsZero() ||
		c.Archived != Any || c.Forks != Any
}

// Select applies c to repos. It returns the matching repos in input order and
// any requested names that don't exist in repos. Name lookup is
// case-insensitive, as GitHub's is.
func Select(repos []github.Repo, c Criteria) (selected []github.Repo, missing []string, err error) {
	if c.Match != "" {
		if _, err := path.Match(c.Match, ""); err != nil {
			return nil, nil, fmt.Errorf("bad --match pattern %q: %w", c.Match, err)
		}
	}

	base := repos
	if len(c.Names) > 0 {
		byName := make(map[string]github.Repo, len(repos))
		for _, r := range repos {
			byName[strings.ToLower(r.Name)] = r
		}
		base = nil
		seen := map[string]bool{}
		for _, n := range c.Names {
			key := strings.ToLower(n)
			if seen[key] {
				continue
			}
			seen[key] = true
			if r, ok := byName[key]; ok {
				base = append(base, r)
			} else {
				missing = append(missing, n)
			}
		}
	}

	for _, r := range base {
		if c.matches(r) {
			selected = append(selected, r)
		}
	}
	return selected, missing, nil
}

func (c Criteria) matches(r github.Repo) bool {
	if c.Match != "" {
		if ok, _ := path.Match(c.Match, r.Name); !ok {
			return false
		}
	}
	if c.Regex != nil && !c.Regex.MatchString(r.Name) {
		return false
	}
	for _, want := range c.Topics {
		if !hasFold(r.Topics, want) {
			return false
		}
	}
	if c.Visibility != "" && !strings.EqualFold(r.Visibility, c.Visibility) {
		return false
	}
	if c.Language != "" && !strings.EqualFold(r.Language, c.Language) {
		return false
	}
	if !c.PushedBefore.IsZero() && !r.PushedAt.Before(c.PushedBefore) {
		return false
	}
	return c.Archived.allows(r.Archived) && c.Forks.allows(r.Fork)
}

func hasFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// ReadNames reads one repo name per line. Blank lines and lines starting with
// # are skipped. A name given as owner/name must belong to owner; the owner
// part is stripped.
func ReadNames(r io.Reader, owner string) ([]string, error) {
	var names []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, err := StripOwner(line, owner)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, sc.Err()
}

// StripOwner turns "owner/name" into "name", and rejects a different owner.
// A bare name passes through.
func StripOwner(s, owner string) (string, error) {
	o, name, ok := strings.Cut(s, "/")
	if !ok {
		return s, nil
	}
	if !strings.EqualFold(o, owner) {
		return "", fmt.Errorf("%q is not owned by %s", s, owner)
	}
	return name, nil
}
