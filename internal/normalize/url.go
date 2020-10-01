// Package normalize provides URL normalization.
//
// https://en.wikipedia.org/wiki/URI_normalization
package normalize

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

// RawURL normalizes the given raw URL.
//
//  - Uppercase percent-encoded triplets.
//  - Lowercase the scheme and hostname.
//  - Lowercase the username.
//  - Decode percent-encoded triplets.
//  - Removes dot segments.
//  - Converts an empty path to `/`.
//  - Removes the default port (:80, :443).
//  - Removes `?` when query is empty.
//  - Remove the fragment.
//
func RawURL(rawurl string) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	return URL(u).String(), nil
}

// URL normalizes a parsed URL.
func URL(u *url.URL) *url.URL {
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = hostname(u)
	u.Path = pathname(u)
	u.RawQuery = query(u.RawQuery)
	u.ForceQuery = false
	u.Fragment = ""
	return u
}

// Hostname normalizes the hostname.
func hostname(u *url.URL) string {
	var host = strings.ToLower(u.Host)

	if j := strings.IndexByte(host, ':'); j != -1 {
		switch port := host[j+1:]; {
		case u.Scheme == "http" && port == "80":
			return host[:j]
		case u.Scheme == "https" && port == "443":
			return host[:j]
		}
	}

	return host
}

// Pathname normalizes the pathname.
func pathname(u *url.URL) string {
	switch u.Path {
	case "", "/":
		return "/"
	default:
		parts := strings.Split(u.Path, "/")
		return path.Join(parts...)
	}
}

// Query sorts the given query.
func query(query string) string {
	if query != "" {
		parts := strings.Split(query, "&")
		sort.Strings(parts)
		return strings.Join(parts, "&")
	}
	return ""
}
