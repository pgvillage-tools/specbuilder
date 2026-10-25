package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("secret")
	c.BaseURL = func(string) string { return srv.URL }
	return c
}

func TestGetRepo(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/org/repo" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing token, got %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"name":"repo","description":"desc","license":{"spdx_id":"MIT"}}`))
	})
	repo, err := c.GetRepo("github.com", "org", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if repo != (Repo{Name: "repo", Description: "desc", License: "MIT"}) {
		t.Errorf("unexpected repo: %+v", repo)
	}
}

func TestGetRepoWithoutLicense(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name":"repo","description":null,"license":null}`))
	})
	repo, err := c.GetRepo("github.com", "org", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if repo.License != "" || repo.Description != "" {
		t.Errorf("unexpected repo: %+v", repo)
	}
}

func TestGetRepoNotFound(t *testing.T) {
	c := newTestClient(t, http.NotFound)
	if _, err := c.GetRepo("github.com", "org", "repo"); err == nil {
		t.Error("expected an error")
	}
}

func TestGetRelease(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/org/repo/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.2-3","body":"log","assets":[
			{"name":"a.tar.gz","browser_download_url":"https://example.com/a.tar.gz"}]}`))
	})
	release, err := c.GetRelease("github.com", "org", "repo", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if release.TagName != "v1.2-3" || len(release.Assets) != 1 || release.Assets[0].URL != "https://example.com/a.tar.gz" {
		t.Errorf("unexpected release: %+v", release)
	}
}

func TestGetReleaseByTag(t *testing.T) {
	for _, release := range []string{"v1.0.0", "tags/v1.0.0"} {
		t.Run(release, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repos/org/repo/releases/tags/v1.0.0" {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
			})
			got, err := c.GetRelease("github.com", "org", "repo", release)
			if err != nil {
				t.Fatal(err)
			}
			if got.TagName != "v1.0.0" {
				t.Errorf("unexpected release: %+v", got)
			}
		})
	}
}

func TestReleasePath(t *testing.T) {
	for release, want := range map[string]string{
		"latest":      "latest",
		"v1.0.0":      "tags/v1.0.0",
		"tags/v1.0.0": "tags/v1.0.0",
	} {
		if got := releasePath(release); got != want {
			t.Errorf("releasePath(%q) = %q, want %q", release, got, want)
		}
	}
}
