package main

import (
	"net/url"
	"strings"
)

// trackingParams are stripped when normalizing a URL for duplicate
// comparison -- they change the URL string without changing what's
// actually being bookmarked.
var trackingParams = []string{
	"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
	"fbclid", "gclid", "mc_cid", "mc_eid", "igshid", "ref",
}

// normalizeForDedupe produces a comparison key for duplicate detection.
// It lowercases scheme and host (case-insensitive per the URL spec),
// strips default ports, a trailing slash, tracking query parameters, and
// the fragment. Path and remaining query VALUES are left exactly as-is
// (they can be case-sensitive on the server), so this only ever collapses
// URLs that are genuinely the same resource, never ones that merely look
// similar.
func normalizeForDedupe(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return strings.TrimSpace(raw)
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	if p := u.Port(); p != "" && !((scheme == "http" && p == "80") || (scheme == "https" && p == "443")) {
		host += ":" + p
	}
	path := strings.TrimSuffix(u.Path, "/")

	query := ""
	if u.RawQuery != "" {
		q := u.Query()
		for _, k := range trackingParams {
			q.Del(k)
		}
		if enc := q.Encode(); enc != "" {
			query = "?" + enc
		}
	}
	return scheme + "://" + host + path + query
}

// findDuplicate returns an existing bookmark whose URL normalizes to the
// same thing as url, or nil if there isn't one.
func findDuplicate(store *Store, url string) *Bookmark {
	target := normalizeForDedupe(url)
	for _, b := range store.Bookmarks {
		if normalizeForDedupe(b.URL) == target {
			return b
		}
	}
	return nil
}
