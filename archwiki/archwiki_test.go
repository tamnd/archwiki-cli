package archwiki

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	start := time.Now()
	_, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestSearchReturnsArticles(t *testing.T) {
	body := `{
		"batchcomplete": "",
		"query": {
			"searchinfo": {"totalhits": 1},
			"search": [{
				"ns": 0,
				"title": "Pacman",
				"pageid": 1234,
				"snippet": "Pacman is a <span class=\"searchmatch\">package</span> manager...",
				"timestamp": "2024-01-15T10:00:00Z"
			}]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	articles, err := c.Search(context.Background(), "pacman", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 1 {
		t.Fatalf("got %d articles, want 1", len(articles))
	}
	a := articles[0]
	if a.Rank != 1 {
		t.Errorf("Rank = %d, want 1", a.Rank)
	}
	if a.Title != "Pacman" {
		t.Errorf("Title = %q, want Pacman", a.Title)
	}
	if a.Snippet == "" {
		t.Error("Snippet is empty")
	}
	if a.Updated != "2024-01-15" {
		t.Errorf("Updated = %q, want 2024-01-15", a.Updated)
	}
	if a.URL != "https://wiki.archlinux.org/title/Pacman" {
		t.Errorf("URL = %q", a.URL)
	}
}

func TestArticleExtract(t *testing.T) {
	body := `{
		"batchcomplete": "",
		"query": {
			"pages": {
				"1234": {
					"pageid": 1234,
					"ns": 0,
					"title": "Pacman",
					"extract": "Pacman is a package manager which tracks installed packages on a local system."
				}
			}
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ext, err := c.Article(context.Background(), "Pacman")
	if err != nil {
		t.Fatal(err)
	}
	if ext.PageID != 1234 {
		t.Errorf("PageID = %d, want 1234", ext.PageID)
	}
	if ext.Title != "Pacman" {
		t.Errorf("Title = %q, want Pacman", ext.Title)
	}
	if ext.Extract == "" {
		t.Error("Extract is empty")
	}
	if ext.URL == "" {
		t.Error("URL is empty")
	}
}

func TestArticleNotFound(t *testing.T) {
	body := `{
		"batchcomplete": "",
		"query": {
			"pages": {
				"-1": {
					"ns": 0,
					"title": "NoSuchPage",
					"pageid": -1,
					"missing": ""
				}
			}
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Article(context.Background(), "NoSuchPage")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestRecentChanges(t *testing.T) {
	body := `{
		"batchcomplete": "",
		"query": {
			"recentchanges": [
				{
					"type": "edit",
					"ns": 0,
					"title": "Systemd",
					"pageid": 5678,
					"user": "WikiEditor",
					"oldlen": 45000,
					"newlen": 45200,
					"timestamp": "2024-01-15T12:30:00Z"
				},
				{
					"type": "edit",
					"ns": 0,
					"title": "Pacman",
					"pageid": 1234,
					"user": "ArchUser",
					"oldlen": 30000,
					"newlen": 29950,
					"timestamp": "2024-01-15T11:00:00Z"
				}
			]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	changes, err := c.Recent(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("got %d changes, want 2", len(changes))
	}
	ch := changes[0]
	if ch.Title != "Systemd" {
		t.Errorf("Title = %q, want Systemd", ch.Title)
	}
	if ch.SizeDiff != 200 {
		t.Errorf("SizeDiff = %d, want 200", ch.SizeDiff)
	}
	if ch.Time != "2024-01-15" {
		t.Errorf("Time = %q, want 2024-01-15", ch.Time)
	}
	if changes[1].SizeDiff != -50 {
		t.Errorf("SizeDiff = %d, want -50", changes[1].SizeDiff)
	}
}

func TestSuggest(t *testing.T) {
	body := `["python",["Python","Python/Tips and tricks","Python package guidelines"],["","",""],["https://wiki.archlinux.org/title/Python","https://wiki.archlinux.org/title/Python/Tips_and_tricks","https://wiki.archlinux.org/title/Python_package_guidelines"]]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	suggestions, err := c.Suggest(context.Background(), "python", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("got %d suggestions, want 3", len(suggestions))
	}
	s := suggestions[0]
	if s.Title != "Python" {
		t.Errorf("Title = %q, want Python", s.Title)
	}
	if s.URL == "" {
		t.Error("URL is empty")
	}
	if s.Rank != 1 {
		t.Errorf("Rank = %d, want 1", s.Rank)
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`plain text`, `plain text`},
		{`<span class="searchmatch">Pacman</span> is a package`, `Pacman is a package`},
		{`&amp; &lt; &gt; &quot; &#39;`, `& < > " '`},
		{`<b>bold</b> and <i>italic</i>`, `bold and italic`},
	}
	for _, tc := range cases {
		got := stripHTML(tc.in)
		if got != tc.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
