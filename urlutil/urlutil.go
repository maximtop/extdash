package urlutil

import (
	"log"
	"net/url"
	"path"
)

func JoinURL(rawBaseURL string, parts ...string) string {
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		log.Panic("Was unable to parse url")
	}

	args := append([]string{baseURL.Path}, parts...)

	baseURL.Path = path.Join(args...)

	return baseURL.String()
}
