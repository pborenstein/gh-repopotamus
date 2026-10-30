package selection

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pborenstein/gh-repopotamus/internal/github"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

var fixture = []github.Repo{
	{Name: "site-blog", Visibility: "PUBLIC", Language: "Go", Topics: []string{"web", "blog"}, PushedAt: day("2024-05-01")},
	{Name: "site-docs", Visibility: "PRIVATE", Language: "JavaScript", Topics: []string{"web"}, PushedAt: day("2026-01-01")},
	{Name: "old-thing", Visibility: "PUBLIC", Archived: true, PushedAt: day("2019-03-03")},
	{Name: "forked-lib", Visibility: "PUBLIC", Fork: true, Language: "go", PushedAt: day("2025-07-07")},
}

func names(repos []github.Repo) []string {
	var out []string
	for _, r := range repos {
		out = append(out, r.Name)
	}
	return out
}

func TestSelect(t *testing.T) {
	tests := []struct {
		name        string
		c           Criteria
		want        []string
		wantMissing []string
	}{
		{"no criteria selects all", Criteria{}, []string{"site-blog", "site-docs", "old-thing", "forked-lib"}, nil},
		{"names keep their order", Criteria{Names: []string{"old-thing", "SITE-BLOG"}}, []string{"old-thing", "site-blog"}, nil},
		{"duplicate names collapse", Criteria{Names: []string{"old-thing", "old-thing"}}, []string{"old-thing"}, nil},
		{"unknown names are missing", Criteria{Names: []string{"nope", "old-thing"}}, []string{"old-thing"}, []string{"nope"}},
		{"glob", Criteria{Match: "site-*"}, []string{"site-blog", "site-docs"}, nil},
		{"regex", Criteria{Regex: regexp.MustCompile(`-(lib|thing)$`)}, []string{"old-thing", "forked-lib"}, nil},
		{"names narrowed by filter", Criteria{Names: []string{"site-blog", "old-thing"}, Match: "site-*"}, []string{"site-blog"}, nil},
		{"topics require all", Criteria{Topics: []string{"web", "blog"}}, []string{"site-blog"}, nil},
		{"visibility", Criteria{Visibility: "private"}, []string{"site-docs"}, nil},
		{"language ignores case", Criteria{Language: "GO"}, []string{"site-blog", "forked-lib"}, nil},
		{"pushed before", Criteria{PushedBefore: day("2025-01-01")}, []string{"site-blog", "old-thing"}, nil},
		{"archived only", Criteria{Archived: Only}, []string{"old-thing"}, nil},
		{"no archived, no forks", Criteria{Archived: Exclude, Forks: Exclude}, []string{"site-blog", "site-docs"}, nil},
		{"forks only", Criteria{Forks: Only}, []string{"forked-lib"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, missing, err := Select(fixture, tt.c)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(names(got), tt.want) {
				t.Errorf("selected = %v, want %v", names(got), tt.want)
			}
			if !reflect.DeepEqual(missing, tt.wantMissing) {
				t.Errorf("missing = %v, want %v", missing, tt.wantMissing)
			}
		})
	}
}

func TestSelectBadGlob(t *testing.T) {
	if _, _, err := Select(fixture, Criteria{Match: "["}); err == nil {
		t.Fatal("want error for bad glob")
	}
}

func TestNarrows(t *testing.T) {
	if (Criteria{Names: []string{"x"}}).Narrows() {
		t.Error("names alone should not count as narrowing")
	}
	if !(Criteria{Forks: Exclude}).Narrows() {
		t.Error("forks filter should count as narrowing")
	}
}

func TestReadNames(t *testing.T) {
	in := "site-blog\n\n# comment\n  pborenstein/old-thing  \n"
	got, err := ReadNames(strings.NewReader(in), "PBorenstein")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"site-blog", "old-thing"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := ReadNames(strings.NewReader("someone-else/x\n"), "pborenstein"); err == nil {
		t.Error("want error for a repo owned by someone else")
	}
}
