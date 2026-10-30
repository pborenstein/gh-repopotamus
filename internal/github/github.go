// Package github wraps the GitHub API calls repopotamus needs.
// Auth comes from gh via go-gh, so there is no token handling here.
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Repo is the metadata used for selecting repos to transfer.
type Repo struct {
	Owner      string
	Name       string
	Visibility string // PUBLIC, PRIVATE, INTERNAL
	Archived   bool
	Fork       bool
	Language   string
	Topics     []string
	PushedAt   time.Time
}

func (r Repo) FullName() string {
	return r.Owner + "/" + r.Name
}

// repositoryOwner resolves to an Organization or a User, so the same query
// works for both kinds of source.
const listReposQuery = `
query($owner: String!, $cursor: String) {
  repositoryOwner(login: $owner) {
    login
    repositories(first: 100, after: $cursor, ownerAffiliations: [OWNER], orderBy: {field: NAME, direction: ASC}) {
      nodes {
        name
        visibility
        isArchived
        isFork
        pushedAt
        primaryLanguage { name }
        repositoryTopics(first: 20) { nodes { topic { name } } }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type listReposResponse struct {
	RepositoryOwner *struct {
		Login        string
		Repositories struct {
			Nodes []struct {
				Name             string
				Visibility       string
				IsArchived       bool
				IsFork           bool
				PushedAt         time.Time
				PrimaryLanguage  *struct{ Name string }
				RepositoryTopics struct {
					Nodes []struct {
						Topic struct{ Name string }
					}
				}
			}
			PageInfo struct {
				HasNextPage bool
				EndCursor   string
			}
		}
	}
}

// ListRepos returns every repo owned by owner (an org or a user).
func ListRepos(owner string) ([]Repo, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}

	var repos []Repo
	vars := map[string]interface{}{"owner": owner, "cursor": nil}
	for {
		var resp listReposResponse
		if err := client.Do(listReposQuery, vars, &resp); err != nil {
			return nil, err
		}
		if resp.RepositoryOwner == nil {
			return nil, fmt.Errorf("owner %q not found", owner)
		}

		page := resp.RepositoryOwner.Repositories
		for _, n := range page.Nodes {
			r := Repo{
				Owner:      resp.RepositoryOwner.Login,
				Name:       n.Name,
				Visibility: n.Visibility,
				Archived:   n.IsArchived,
				Fork:       n.IsFork,
				PushedAt:   n.PushedAt,
			}
			if n.PrimaryLanguage != nil {
				r.Language = n.PrimaryLanguage.Name
			}
			for _, t := range n.RepositoryTopics.Nodes {
				r.Topics = append(r.Topics, t.Topic.Name)
			}
			repos = append(repos, r)
		}

		if !page.PageInfo.HasNextPage {
			return repos, nil
		}
		vars["cursor"] = page.PageInfo.EndCursor
	}
}

// Transfer asks GitHub to move owner/repo to newOwner. GitHub answers 202 and
// completes the move asynchronously.
func Transfer(owner, repo, newOwner string) error {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]string{"new_owner": newOwner})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("repos/%s/%s/transfer", owner, repo)
	return client.Post(path, bytes.NewReader(body), nil)
}
