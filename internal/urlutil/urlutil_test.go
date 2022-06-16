package urlutil_test

import (
	"net/url"
	"testing"

	"github.com/maximtop/extdash/internal/urlutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinURL(t *testing.T) {
	t.Run("Sequential joins do not update same url", func(t *testing.T) {
		const rawBaseURL = "https://example.org"
		baseURL, err := url.Parse(rawBaseURL)
		require.NoError(t, err)

		part := "test"
		joinedURL := urlutil.JoinURL(baseURL, part)
		assert.Equal(t, rawBaseURL+"/"+part, joinedURL)

		part2 := "test2"
		secondJoinedURL := urlutil.JoinURL(baseURL, part2)
		assert.Equal(t, rawBaseURL+"/"+part2, secondJoinedURL)
	})
}
