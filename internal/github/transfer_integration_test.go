//go:build integration

// Round-trip transfer against real GitHub. Moves a throwaway repo to another
// owner and back. Run with:
//
//	REPOPOTAMUS_TEST_REPO=pborenstein/test-repo \
//	REPOPOTAMUS_TEST_DEST=documentation-rodeo \
//	go test -tags integration -v ./internal/github/
package github

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestTransferRoundTrip(t *testing.T) {
	full := os.Getenv("REPOPOTAMUS_TEST_REPO")
	dest := os.Getenv("REPOPOTAMUS_TEST_DEST")
	if full == "" || dest == "" {
		t.Skip("set REPOPOTAMUS_TEST_REPO and REPOPOTAMUS_TEST_DEST")
	}
	src, name, ok := strings.Cut(full, "/")
	if !ok {
		t.Fatalf("REPOPOTAMUS_TEST_REPO must be owner/repo, got %q", full)
	}

	t.Logf("%s/%s -> %s", src, name, dest)
	if err := Transfer(src, name, dest); err != nil {
		t.Fatalf("transfer out: %v", err)
	}
	waitForOwner(t, name, dest)

	// The repo resolves at its new owner before GitHub finishes processing the
	// transfer. A second transfer in that window gets 422 "Repository.data is
	// invalid", so retry until it is accepted.
	t.Logf("%s/%s -> %s", dest, name, src)
	deadline := time.Now().Add(60 * time.Second)
	for {
		err := Transfer(dest, name, src)
		if err == nil {
			break
		}
		var httpErr *api.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 422 || time.Now().After(deadline) {
			t.Fatalf("transfer back: %v", err)
		}
		t.Logf("transfer back not accepted yet (422), retrying")
		time.Sleep(5 * time.Second)
	}
	waitForOwner(t, name, src)
}

// waitForOwner polls until owner/name resolves and reports owner as its owner.
// Transfers complete asynchronously after the API returns 202.
func waitForOwner(t *testing.T, name, owner string) {
	t.Helper()
	client, err := api.DefaultRESTClient()
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		var resp struct {
			Owner struct{ Login string }
		}
		err := client.Get("repos/"+owner+"/"+name, &resp)
		if err == nil && strings.EqualFold(resp.Owner.Login, owner) {
			t.Logf("%s/%s is in place", owner, name)
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("%s/%s did not appear within 60s", owner, name)
}
