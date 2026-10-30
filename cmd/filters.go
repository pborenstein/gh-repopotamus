package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pborenstein/gh-repopotamus/internal/selection"
	"github.com/spf13/pflag"
)

// filterFlags are the selection filters shared by list and move.
type filterFlags struct {
	match        string
	regex        string
	topics       []string
	visibility   string
	language     string
	pushedBefore string
	archived     bool
	noArchived   bool
	forks        bool
	noForks      bool
}

func (f *filterFlags) register(fs *pflag.FlagSet) {
	fs.StringVar(&f.match, "match", "", "glob matched against the repo name, e.g. 'site-*'")
	fs.StringVar(&f.regex, "regex", "", "regular expression matched against the repo name")
	fs.StringSliceVar(&f.topics, "topic", nil, "repo has this topic (repeatable; all must match)")
	fs.StringVar(&f.visibility, "visibility", "", "public, private or internal")
	fs.StringVar(&f.language, "language", "", "primary language")
	fs.StringVar(&f.pushedBefore, "pushed-before", "", "last push before DATE (YYYY-MM-DD)")
	fs.BoolVar(&f.archived, "archived", false, "only archived repos")
	fs.BoolVar(&f.noArchived, "no-archived", false, "exclude archived repos")
	fs.BoolVar(&f.forks, "forks", false, "only forks")
	fs.BoolVar(&f.noForks, "no-forks", false, "exclude forks")
}

func (f *filterFlags) criteria() (selection.Criteria, error) {
	var c selection.Criteria
	c.Match = f.match
	c.Topics = f.topics
	c.Language = f.language

	if f.regex != "" {
		re, err := regexp.Compile(f.regex)
		if err != nil {
			return c, fmt.Errorf("bad --regex: %w", err)
		}
		c.Regex = re
	}

	switch v := strings.ToLower(f.visibility); v {
	case "", "public", "private", "internal":
		c.Visibility = v
	default:
		return c, fmt.Errorf("--visibility must be public, private or internal, got %q", f.visibility)
	}

	if f.pushedBefore != "" {
		t, err := time.Parse("2006-01-02", f.pushedBefore)
		if err != nil {
			return c, fmt.Errorf("--pushed-before must be YYYY-MM-DD, got %q", f.pushedBefore)
		}
		c.PushedBefore = t
	}

	var err error
	if c.Archived, err = tri(f.archived, f.noArchived, "archived"); err != nil {
		return c, err
	}
	if c.Forks, err = tri(f.forks, f.noForks, "forks"); err != nil {
		return c, err
	}
	return c, nil
}

func tri(only, exclude bool, name string) (selection.Tri, error) {
	switch {
	case only && exclude:
		return selection.Any, fmt.Errorf("--%s and --no-%s can't be combined", name, name)
	case only:
		return selection.Only, nil
	case exclude:
		return selection.Exclude, nil
	}
	return selection.Any, nil
}
