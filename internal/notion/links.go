package notion

import (
	"net/url"
	"regexp"
	"strings"
)

// LinkKind says what a link in a page leads to.
type LinkKind int

const (
	LinkPage     LinkKind = iota // Another Notion page
	LinkDatabase                 // A Notion database
	LinkWeb                      // Anything else on the web
)

// PageLink is something in a page that can be followed.
type PageLink struct {
	Kind  LinkKind
	Title string
	URL   string
	// ID is the Notion ID for pages and databases; empty for web links.
	ID string
}

var (
	blockLinkOpen = regexp.MustCompile(`^<(page|database)(\s[^>]*)?>(.*)$`)
	// A mention is either self-closing or wraps its title
	mentionLink = regexp.MustCompile(`<(mention[\w-]*)(\s[^>]*?)?(?:/>|>(.*?)</mention[\w-]*>)`)
	// The URL may hold one level of parentheses, as Wikipedia links do
	markdownURLLink = regexp.MustCompile(`(!?)\[([^\]]*)\]\(((?:[^()\s]|\([^()\s]*\))+)\)`)
	notionIDPattern = regexp.MustCompile(`([0-9a-fA-F]{32})$`)
	bareIDPattern   = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
	codeFenceOpen   = regexp.MustCompile("^\\s*(`{3,}|~{3,})(.*)$")
)

// PageLinks lists what can be followed from a page's Notion-flavored
// Markdown, in document order and without repeats: child pages, databases,
// mentions of pages, and ordinary links. Code blocks and images are skipped.
func PageLinks(markdown string) []PageLink {
	var links []PageLink
	seen := map[string]bool{}
	add := func(link PageLink) {
		link.Title = strings.TrimSpace(link.Title)
		key := link.URL
		if link.ID != "" {
			key = link.ID
		}
		if link.URL == "" || seen[key] {
			return
		}
		if link.Title == "" {
			link.Title = link.URL
		}
		seen[key] = true
		links = append(links, link)
	}

	var pending *PageLink // An open <page>/<database> tag waiting for its title
	fence := ""
	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(raw)

		if fence != "" {
			if len(line) >= len(fence) && strings.Trim(line, fence[:1]) == "" {
				fence = ""
			}
			continue
		}
		if m := codeFenceOpen.FindStringSubmatch(line); m != nil && (m[1][0] != '`' || !strings.Contains(m[2], "`")) {
			fence = m[1]
			continue
		}

		if pending != nil {
			// The title is the next line with text on it, and nothing more:
			// a tag that is never closed must not swallow the rest of the page.
			if line == "" {
				continue
			}
			link := *pending
			pending = nil
			switch {
			case strings.HasPrefix(line, "</"):
				add(link)
				continue
			case !blockLinkOpen.MatchString(line):
				title, _, _ := strings.Cut(line, "</")
				link.Title = inlineMarkdown(title)
				add(link)
				continue
			}
			add(link) // Another tag opens: this one had no title
		}

		if m := blockLinkOpen.FindStringSubmatch(line); m != nil {
			link := blockLink(m[1], attr(m[2], "url"))
			rest := m[3]
			if end := strings.Index(rest, "</"); end >= 0 {
				link.Title = inlineMarkdown(rest[:end])
				add(link)
			} else if title := inlineMarkdown(rest); strings.TrimSpace(title) != "" {
				link.Title = title
				add(link)
			} else {
				pending = &link
			}
			continue
		}

		for _, m := range mentionLink.FindAllStringSubmatch(line, -1) {
			target := attr(m[2], "url")
			link := classifyURL(target)
			if strings.Contains(m[1], "database") || strings.Contains(m[1], "data-source") {
				link.Kind = LinkDatabase
			}
			link.Title = inlineMarkdown(m[3])
			add(link)
		}
		for _, m := range markdownURLLink.FindAllStringSubmatch(line, -1) {
			if m[1] == "!" {
				continue // An image, not a link
			}
			link := classifyURL(m[3])
			link.Title = inlineMarkdown(m[2])
			add(link)
		}
	}
	if pending != nil {
		add(*pending)
	}
	return links
}

// blockLink describes a <page> or <database> tag. A tag whose URL holds no
// Notion ID cannot be opened in the app, so it is treated as a web link, or
// dropped if it is not one either.
func blockLink(tag, target string) PageLink {
	id := notionID(target)
	if id == "" {
		return classifyURL(target)
	}
	link := PageLink{Kind: LinkPage, URL: target, ID: id}
	if tag == "database" {
		link.Kind = LinkDatabase
	}
	return link
}

// classifyURL decides whether a URL leads to a Notion page or elsewhere.
// Only http and https links are kept.
func classifyURL(target string) PageLink {
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return PageLink{}
	}
	link := PageLink{Kind: LinkWeb, URL: target}
	if isNotionHost(parsed.Host) {
		if id := notionID(target); id != "" {
			link.Kind = LinkPage
			link.ID = id
		}
	}
	return link
}

func isNotionHost(host string) bool {
	host = strings.ToLower(host)
	for _, domain := range []string{"notion.so", "notion.com", "notion.site"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

// notionID extracts the 32-character ID that ends the path of a Notion URL
// ("https://app.notion.com/p/<id>", "https://www.notion.so/Title-<id>"). A
// row opened in a side peek ("...?p=<row id>") is the row, not the view
// behind it.
func notionID(target string) string {
	parsed, err := url.Parse(target)
	if err != nil || !isNotionHost(parsed.Host) {
		return ""
	}
	if peek := parsed.Query().Get("p"); bareIDPattern.MatchString(peek) {
		return strings.ToLower(peek)
	}
	path := strings.TrimRight(parsed.Path, "/")
	if m := notionIDPattern.FindStringSubmatch(path); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}
