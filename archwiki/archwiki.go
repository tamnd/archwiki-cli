// Package archwiki is the library behind the archwiki command: the HTTP client,
// request shaping, and the typed data models for the Arch Linux Wiki.
//
// The Arch Linux Wiki runs on MediaWiki and exposes a public JSON API at
// https://wiki.archlinux.org/api.php. No API key or authentication is required.
package archwiki

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to Arch Linux Wiki.
const DefaultUserAgent = "archwiki/dev (+https://github.com/tamnd/archwiki-cli)"

// ErrNotFound is returned when an article does not exist.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters for Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults for talking to wiki.archlinux.org.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://wiki.archlinux.org",
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Arch Linux Wiki MediaWiki API.
type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		baseURL:    cfg.BaseURL,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// ─── HTTP layer ───────────────────────────────────────────────────────────────

// get fetches rawURL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(body)) == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// apiURL builds a full API URL from action parameters.
func (c *Client) apiURL(params url.Values) string {
	params.Set("format", "json")
	return c.baseURL + "/api.php?" + params.Encode()
}

// ─── API methods ─────────────────────────────────────────────────────────────

// Search searches ArchWiki articles for query and returns up to limit results.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "search")
	params.Set("srsearch", query)
	params.Set("srlimit", strconv.Itoa(limit))
	params.Set("srprop", "snippet|timestamp")
	params.Set("srnamespace", "0")

	var resp searchResp
	if err := c.getJSON(ctx, c.apiURL(params), &resp); err != nil {
		return nil, err
	}

	out := make([]Article, 0, len(resp.Query.Search))
	for i, h := range resp.Query.Search {
		out = append(out, Article{
			Rank:    i + 1,
			Title:   h.Title,
			Snippet: stripHTML(h.Snippet),
			Updated: isoDate(h.Timestamp),
			URL:     articleURL(h.Title),
		})
	}
	return out, nil
}

// Article fetches the intro extract of the named article.
// Returns ErrNotFound when the page does not exist.
func (c *Client) Article(ctx context.Context, title string) (Extract, error) {
	params := url.Values{}
	params.Set("action", "query")
	params.Set("titles", title)
	params.Set("prop", "extracts")
	params.Set("exintro", "1")
	params.Set("explaintext", "1")
	params.Set("redirects", "1")

	var resp extractResp
	if err := c.getJSON(ctx, c.apiURL(params), &resp); err != nil {
		return Extract{}, err
	}

	for _, page := range resp.Query.Pages {
		if page.PageID == -1 {
			return Extract{}, ErrNotFound
		}
		return Extract{
			PageID:  page.PageID,
			Title:   page.Title,
			Extract: strings.TrimSpace(page.Extract),
			URL:     articleURL(page.Title),
		}, nil
	}
	return Extract{}, ErrNotFound
}

// Recent returns up to limit recent changes in the main namespace.
func (c *Client) Recent(ctx context.Context, limit int) ([]Change, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "recentchanges")
	params.Set("rclimit", strconv.Itoa(limit))
	params.Set("rcprop", "title|timestamp|user|sizes")
	params.Set("rcnamespace", "0")

	var resp recentResp
	if err := c.getJSON(ctx, c.apiURL(params), &resp); err != nil {
		return nil, err
	}

	out := make([]Change, 0, len(resp.Query.RecentChanges))
	for i, h := range resp.Query.RecentChanges {
		out = append(out, Change{
			Rank:     i + 1,
			Title:    h.Title,
			User:     h.User,
			Time:     isoDate(h.Timestamp),
			SizeDiff: h.NewLen - h.OldLen,
			URL:      articleURL(h.Title),
		})
	}
	return out, nil
}

// Suggest returns up to limit autocomplete suggestions for prefix using OpenSearch.
func (c *Client) Suggest(ctx context.Context, prefix string, limit int) ([]Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	params.Set("action", "opensearch")
	params.Set("search", prefix)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("namespace", "0")

	var raw openSearchResp
	if err := c.getJSON(ctx, c.apiURL(params), &raw); err != nil {
		return nil, err
	}

	var titles []string
	var urls []string
	if err := json.Unmarshal(raw[1], &titles); err != nil {
		return nil, fmt.Errorf("opensearch titles: %w", err)
	}
	if err := json.Unmarshal(raw[3], &urls); err != nil {
		return nil, fmt.Errorf("opensearch urls: %w", err)
	}

	n := len(titles)
	if len(urls) < n {
		n = len(urls)
	}
	out := make([]Suggestion, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, Suggestion{
			Rank:  i + 1,
			Title: titles[i],
			URL:   urls[i],
		})
	}
	return out, nil
}
