package urlutil

import (
	"net/url"
	"path"
)

// JoinURL appends parts to baseURL and returns joined fullURL
func JoinURL(baseURL *url.URL, parts ...string) string {
	copyURL := *baseURL
	args := append([]string{copyURL.Path}, parts...)
	copyURL.Path = path.Join(args...)
	return copyURL.String()
}
