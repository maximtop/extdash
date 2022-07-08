package edge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/errors"

	"github.com/AdguardTeam/golibs/log"
	"github.com/maximtop/extdash/internal/urlutil"
)

const requestTimeout = 5 * time.Minute

// Client represent the edge client.
type Client struct {
	ClientID       string
	ClientSecret   string
	AccessTokenURL *url.URL
}

// NewClient creates a new edge Client instance.
func NewClient(clientID, clientSecret, rawAccessTokenURL string) (client Client, err error) {
	accessTokenURL, err := url.Parse(rawAccessTokenURL)
	if err != nil {
		return Client{}, fmt.Errorf("failed to parse access token URL: %w", err)
	}

	return Client{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		AccessTokenURL: accessTokenURL,
	}, nil
}

type AuthorizeResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

// Authorize returns the access token.
func (c *Client) Authorize() (accessToken string, err error) {
	form := url.Values{
		"client_id":     {c.ClientID},
		"scope":         {"https://api.addons.microsoftedge.microsoft.com/.default"},
		"client_secret": {c.ClientSecret},
		"grant_type":    {"client_credentials"},
	}

	req, err := http.NewRequest(http.MethodPost, c.AccessTokenURL.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("[Authorize] failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: requestTimeout}

	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("[Authorize] failed to send request: %w", err)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("[Authorize] failed to read response: %w", err)
	}

	var authorizeResponse AuthorizeResponse

	err = json.Unmarshal(responseBody, &authorizeResponse)
	if err != nil {
		return "", fmt.Errorf("[Authorize] failed to unmarshal response: %s, due to error %w", responseBody, err)
	}

	return authorizeResponse.AccessToken, nil
}

// Store represents the edge store instance
type Store struct {
	URL *url.URL
}

// NewStore creates a new edge Store instance.
func NewStore(rawURL string) (store Store, err error) {
	URL, err := url.Parse(rawURL)
	if err != nil {
		return Store{}, fmt.Errorf("[NewStore] failed to parse URL: %s due to error: %w", rawURL, err)
	}

	return Store{
		URL: URL,
	}, nil
}

// Status represents the status of the update or publish.
type Status int64

const (
	InProgress Status = iota
	Succeeded
	Failed
)

// String returns the string representation of the status.
func (u Status) String() string {
	switch u {
	case InProgress:
		return "InProgress"
	case Succeeded:
		return "Succeeded"
	case Failed:
		return "Failed"
	}

	return "unknown"
}

type StatusError struct {
	Message string `json:"message"`
}

type UploadStatusResponse struct {
	ID              string        `json:"id"`
	CreatedTime     string        `json:"createdTime"`
	LastUpdatedTime string        `json:"lastUpdatedTime"`
	Status          string        `json:"status"`
	Message         string        `json:"message"`
	ErrorCode       string        `json:"errorCode"`
	Errors          []StatusError `json:"errors"`
}

type UpdateOptions struct {
	RetryTimeout      time.Duration
	WaitStatusTimeout time.Duration
}

// Update uploads the update to the store and waits for the update to be processed.
func (s Store) Update(c Client, appID, filepath string, updateOptions UpdateOptions) (result *UploadStatusResponse, err error) {
	const defaultRetryTimeout = 5 * time.Second
	const defaultWaitStatusTimeout = 1 * time.Minute

	if updateOptions.RetryTimeout == 0 {
		updateOptions.RetryTimeout = defaultRetryTimeout
	}

	if updateOptions.WaitStatusTimeout == 0 {
		updateOptions.WaitStatusTimeout = defaultWaitStatusTimeout
	}

	operationID, err := s.UploadUpdate(c, appID, filepath)
	if err != nil {
		return nil, fmt.Errorf(
			"[Update] failed to upload update for appID: %s, with filepath: %s, due to error: %w", appID, filepath, err,
		)
	}

	startTime := time.Now()

	for {
		if time.Now().After(startTime.Add(updateOptions.WaitStatusTimeout)) {
			return nil, fmt.Errorf("update failed due to timeout")
		}

		log.Debug("getting upload status...")

		status, err := s.UploadStatus(c, appID, operationID)
		if err != nil {
			return nil, fmt.Errorf(
				"[Update] failed to get upload status for appID: %s, with operationID: %s, due to error: %w", appID, operationID, err,
			)
		}

		if status.Status == InProgress.String() {
			log.Debug("update is in progress, retry in: %s", updateOptions.RetryTimeout)
			time.Sleep(updateOptions.RetryTimeout)

			continue
		}

		if status.Status == Succeeded.String() {
			return status, nil
		}

		if status.Status == Failed.String() {
			return nil, fmt.Errorf("update failed due to %s, full error %+v", status.Message, status)
		}
	}
}

// UploadUpdate uploads the update to the store.
func (s Store) UploadUpdate(c Client, appID, filepath string) (result string, err error) {
	const apiPath = "/v1/products"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "submissions/draft/package")

	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("[UploadUpdate] failed to open file: %s due to error: %w", filepath, err)
	}
	defer func() {
		err := errors.WithDeferred(err, file.Close())
		if err != nil {
			log.Debug("[UploadUpdate] failed to close file: %s due to error: %s", filepath, err)
		}
	}()

	req, err := http.NewRequest(http.MethodPost, apiURL, file)
	if err != nil {
		return "", fmt.Errorf("[UploadUpdate] failed to create request: %w", err)
	}

	accessToken, err := c.Authorize()
	if err != nil {
		return "", fmt.Errorf("[UploadUpdate] failed to get access token: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/zip")

	client := http.Client{Timeout: requestTimeout}

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("[UploadUpdate] failed to send request: %w", err)
	}

	if res.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("[UploadUpdate] received wrong response %s", res.Status)
	}

	operationID := res.Header.Get("Location")

	if operationID == "" {
		return "", fmt.Errorf("[UploadUpdate] received empty operation ID")
	}

	return operationID, nil
}

// UploadStatus returns the status of the upload.
func (s Store) UploadStatus(c Client, appID, operationID string) (response *UploadStatusResponse, err error) {
	apiPath := "v1/products"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "submissions/draft/package/operations", operationID)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("[UploadStatus] failed to create request: %w", err)
	}

	accessToken, err := c.Authorize()
	if err != nil {
		return nil, fmt.Errorf("[UploadStatus] failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := http.Client{
		Timeout: requestTimeout,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[UploadStatus] failed to send request: %w", err)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("[UploadStatus] failed to read response body: %w", err)
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("[UploadStatus] failed to unmarshal response body: %s, due to error %w", responseBody, err)
	}

	return response, nil
}

// PublishExtension publishes the extension to the store and returns operationID.
func (s Store) PublishExtension(c Client, appID string) (result string, err error) {
	apiPath := "/v1/products/"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "submissions")

	// TODO (maximtop): consider adding body to the request with notes for reviewers.
	req, err := http.NewRequest(http.MethodPost, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("[PublishExtension] failed to create request: %w", err)
	}

	accessToken, err := c.Authorize()
	if err != nil {
		return "", fmt.Errorf("[PublishExtension] failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := http.Client{Timeout: requestTimeout}

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("[PublishExtension] failed to send request: %w", err)
	}

	if res.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("received wrong response %s", res.Status)
	}

	operationID := res.Header.Get("Location")

	if operationID == "" {
		return "", fmt.Errorf("received empty operation ID")
	}

	return operationID, nil
}

type PublishStatusResponse struct {
	ID              string        `json:"id"`
	CreatedTime     string        `json:"createdTime"`
	LastUpdatedTime string        `json:"lastUpdatedTime"`
	Status          string        `json:"status"`
	Message         string        `json:"message"`
	ErrorCode       string        `json:"errorCode"`
	Errors          []StatusError `json:"errors"`
}

// PublishStatus returns the status of the extension publish.
func (s Store) PublishStatus(c Client, appID, operationID string) (response *PublishStatusResponse, err error) {
	apiPath := "v1/products/"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "submissions/operations", operationID)

	accessToken, err := c.Authorize()
	if err != nil {
		return nil, fmt.Errorf("[PublishStatus] failed to get access token: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("[PublishStatus] failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := http.Client{Timeout: requestTimeout}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[PublishStatus] failed to send request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[PublishStatus] received wrong response %s", res.Status)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("[PublishStatus] failed to read response body: %w", err)
	}

	response = &PublishStatusResponse{}
	err = json.Unmarshal(responseBody, response)
	if err != nil {
		return nil, fmt.Errorf("[PublishStatus] failed to unmarshal response body: %s, due to error %w", responseBody, err)
	}

	if response.Status == Failed.String() {
		return nil, fmt.Errorf("publish failed due to \"%s\", full error: %+v", response.Message, response)
	}

	return response, nil
}

// Publish publishes the extension to the store.
func (s Store) Publish(c Client, appID string) (response *PublishStatusResponse, err error) {
	operationID, err := s.PublishExtension(c, appID)
	if err != nil {
		return nil, fmt.Errorf("[Publish] failed to publish extension with appID: %s, due to error: %w", appID, err)
	}

	return s.PublishStatus(c, appID, operationID)
}
