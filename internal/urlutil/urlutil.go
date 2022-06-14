package urlutil

import (
	"net/url"
	"path"
)

// JoinURL appends parts to baseURL and returns joined fullURL
func JoinURL(baseURL *url.URL, parts ...string) string {
	args := append([]string{baseURL.Path}, parts...)
	baseURL.Path = path.Join(args...)
	return baseURL.String()
}
