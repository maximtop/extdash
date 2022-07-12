package chrome

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/maximtop/extdash/internal/fileutil"
	"github.com/maximtop/extdash/internal/urlutil"
)

// Client describes structure of the client.
type Client struct {
	URL          string
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// maxReadLimit limits response size returned from the store.
const maxReadLimit = 10 * fileutil.MB

type AuthorizeResponse struct {
	AccessToken string `json:"access_token"`
}

// Authorize retrieves access token.
func (c *Client) Authorize() (accessToken string, err error) {
	data := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"refresh_token": {c.RefreshToken},
		"grant_type":    {"refresh_token"},
		"redirect_uri":  {"urn:ietf:wg:oauth:2.0:oob"},
	}

	res, err := http.PostForm(c.URL, data)
	if err != nil {
		return "", fmt.Errorf("posting a form: %w", err)
	}

	defer func() { err = errors.WithDeferred(err, res.Body.Close()) }()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	result := &AuthorizeResponse{}

	err = json.Unmarshal(body, result)
	if err != nil {
		return "", fmt.Errorf("unmarshaling response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got code %d, body: %q", res.StatusCode, body)
	}

	return result.AccessToken, nil
}

// Store describes structure of the store.
type Store struct {
	URL *url.URL
}

// NewStore parses url and creates new store instance.
func NewStore(rawURL string) (s Store, err error) {
	URL, err := url.Parse(rawURL)
	if err != nil {
		return Store{}, fmt.Errorf("error parsing url: %s due to: %w", rawURL, err)
	}

	return Store{URL: URL}, nil
}

// StatusResponse describes status response fields.
type StatusResponse struct {
	Kind        string
	ID          string
	PublicKey   string
	UploadState string
	CrxVersion  string
}

const requestTimeout = 5 * time.Minute

// Status retrieves status of the extension in the store.
func (s *Store) Status(c Client, appID string) (result *StatusResponse, err error) {
	const apiPath = "chromewebstore/v1.1/items"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID)

	accessToken, err := c.Authorize()
	if err != nil {
		return nil, fmt.Errorf("getting access token: %w", err)
	}

	client := &http.Client{Timeout: requestTimeout}

	var req *http.Request

	req, err = http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	q := req.URL.Query()
	q.Add("projection", "DRAFT")
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	defer func() { err = errors.WithDeferred(err, res.Body.Close()) }()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, body)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling response body: %w", err)
	}

	return result, nil
}

// InsertResponse describes structure returned on the insert request.
type InsertResponse struct {
	Kind        string
	ID          string
	UploadState string
}

// Insert uploads a package to create a new store item.
func (s *Store) Insert(c Client, filePath string) (result *InsertResponse, err error) {
	const apiPath = "upload/chromewebstore/v1.1/items"
	apiURL := urlutil.JoinURL(s.URL, apiPath)

	accessToken, err := c.Authorize()
	if err != nil {
		return nil, fmt.Errorf("getting access token: %w", err)
	}

	body, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	client := &http.Client{Timeout: requestTimeout}

	req, err := http.NewRequest(http.MethodPost, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, res.Body.Close()) }()

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, responseBody)
	}

	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling response body: %w", err)
	}

	return result, nil
}

// UpdateResponse describes response returned on update request.
type UpdateResponse struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	UploadState string `json:"uploadState"`
}

// Update uploads new version of the package to the store.
func (s *Store) Update(c Client, appID, filePath string) (result *UpdateResponse, err error) {
	const apiPath = "upload/chromewebstore/v1.1/items/"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID)

	accessToken, err := c.Authorize()
	if err != nil {
		return nil, fmt.Errorf("getting access token: %w", err)
	}

	client := &http.Client{Timeout: requestTimeout}

	body, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, res.Body.Close()) }()

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, responseBody)
	}

	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling response body: %w", err)
	}

	return result, nil
}

// PublishResponse describes response returned on publish request.
type PublishResponse struct {
	Kind         string   `json:"kind"`
	ItemID       string   `json:"item_id"`
	Status       []string `json:"status"`
	StatusDetail []string `json:"statusDetail"`
}

// Publish publishes app to the store.
func (s *Store) Publish(c Client, appID string) (result *PublishResponse, err error) {
	const apiPath = "chromewebstore/v1.1/items"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "publish")

	accessToken, err := c.Authorize()
	if err != nil {
		return nil, fmt.Errorf("getting access token: %w", err)
	}

	client := &http.Client{Timeout: requestTimeout}

	req, err := http.NewRequest(http.MethodPost, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, res.Body.Close()) }()

	resultBody, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, resultBody)
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling response body: %w", err)
	}

	return result, nil
}