package archwiki

import (
	"encoding/json"
	"strings"
)

// Article is the record emitted for search results.
type Article struct {
	Rank    int    `json:"rank"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Updated string `json:"updated"`
	URL     string `json:"url"`
}

// Change is the record emitted for recent wiki changes.
type Change struct {
	Rank     int    `json:"rank"`
	Title    string `json:"title"`
	User     string `json:"user"`
	Time     string `json:"time"`
	SizeDiff int    `json:"size_diff"`
	URL      string `json:"url"`
}

// Suggestion is the record emitted for opensearch autocomplete.
type Suggestion struct {
	Rank  int    `json:"rank"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Extract is the record emitted for article text.
type Extract struct {
	PageID  int    `json:"page_id"`
	Title   string `json:"title"`
	Extract string `json:"extract"`
	URL     string `json:"url"`
}

// Category is the record emitted by the categories command.
type Category struct {
	Rank int    `json:"rank"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// MainCategories is the hardcoded list returned by the categories command.
var MainCategories = []Category{
	{1, "Installation", "https://wiki.archlinux.org/title/Category:Installation"},
	{2, "Getting and installing Arch", "https://wiki.archlinux.org/title/Category:Getting_and_installing_Arch"},
	{3, "System administration", "https://wiki.archlinux.org/title/Category:System_administration"},
	{4, "Package management", "https://wiki.archlinux.org/title/Category:Package_management"},
	{5, "Networking", "https://wiki.archlinux.org/title/Category:Networking"},
	{6, "Security", "https://wiki.archlinux.org/title/Category:Security"},
	{7, "X Window System", "https://wiki.archlinux.org/title/Category:X_Window_System"},
	{8, "Graphical user interfaces", "https://wiki.archlinux.org/title/Category:Graphical_user_interfaces"},
	{9, "Multimedia", "https://wiki.archlinux.org/title/Category:Multimedia"},
	{10, "Development", "https://wiki.archlinux.org/title/Category:Development"},
	{11, "Hardware", "https://wiki.archlinux.org/title/Category:Hardware"},
	{12, "Virtualization", "https://wiki.archlinux.org/title/Category:Virtualization"},
	{13, "Servers", "https://wiki.archlinux.org/title/Category:Servers"},
	{14, "Arch Linux", "https://wiki.archlinux.org/title/Category:Arch_Linux"},
	{15, "English", "https://wiki.archlinux.org/title/Category:English"},
}

// ─── wire types ──────────────────────────────────────────────────────────────

type searchResp struct {
	Query struct {
		Search []searchHit `json:"search"`
	} `json:"query"`
}

type searchHit struct {
	NS        int    `json:"ns"`
	Title     string `json:"title"`
	PageID    int    `json:"pageid"`
	Snippet   string `json:"snippet"`
	Timestamp string `json:"timestamp"`
}

type extractResp struct {
	Query struct {
		Pages map[string]extractPage `json:"pages"`
	} `json:"query"`
}

type extractPage struct {
	PageID  int    `json:"pageid"`
	NS      int    `json:"ns"`
	Title   string `json:"title"`
	Extract string `json:"extract"`
}

type recentResp struct {
	Query struct {
		RecentChanges []recentHit `json:"recentchanges"`
	} `json:"query"`
}

type recentHit struct {
	Type      string `json:"type"`
	NS        int    `json:"ns"`
	Title     string `json:"title"`
	PageID    int    `json:"pageid"`
	User      string `json:"user"`
	OldLen    int    `json:"oldlen"`
	NewLen    int    `json:"newlen"`
	Timestamp string `json:"timestamp"`
}

// openSearchResp holds the raw 4-element opensearch array.
// [0]=query, [1]=titles, [2]=descriptions, [3]=urls
type openSearchResp [4]json.RawMessage

// ─── helpers ─────────────────────────────────────────────────────────────────

// articleURL returns the canonical ArchWiki URL for a page title.
func articleURL(title string) string {
	return "https://wiki.archlinux.org/title/" + strings.ReplaceAll(title, " ", "_")
}

// stripHTML removes HTML tags from s and decodes common entities.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.ReplaceAll(out, "&amp;", "&")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&quot;", `"`)
	out = strings.ReplaceAll(out, "&#39;", "'")
	out = strings.ReplaceAll(out, "&apos;", "'")
	return strings.TrimSpace(out)
}

// isoDate formats a MediaWiki timestamp (RFC3339) as "2006-01-02".
// Returns the original string if parsing fails.
func isoDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
