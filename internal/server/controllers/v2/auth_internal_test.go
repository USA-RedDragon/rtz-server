package v2

import (
	"net/url"
	"testing"
)

func TestGitHubTokenURLEscapes(t *testing.T) {
	t.Parallel()
	urldata := url.Values{}
	urldata.Set("code", "a&client_id=evil")
	urldata.Set("client_id", "id")
	urldata.Set("client_secret", "s#cret")

	u, err := url.Parse(githubTokenURL(urldata))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := u.Query()
	if q.Get("code") != "a&client_id=evil" || q.Get("client_id") != "id" || q.Get("client_secret") != "s#cret" {
		t.Errorf("unexpected query: %v", q)
	}
}
