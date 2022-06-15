package edge

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/maximtop/extdash/internal/urlutil"
)

const requestTimeout = 5 * time.Minute

type Client struct {
	ClientID       string
	ClientSecret   string
	AccessTokenURL *url.URL
}

func NewClient(clientID, clientSecret, rawAccessTokenURL string) (client Client, err error) {
	accessTokenURL, err := url.Parse(rawAccessTokenURL)
	if err != nil {
		return Client{}, err
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

func (c *Client) Authorize() (accessToken string, err error) {
	form := url.Values{
		"client_id":     {c.ClientID},
		"scope":         {"https://api.addons.microsoftedge.microsoft.com/.default"},
		"client_secret": {c.ClientSecret},
		"grant_type":    {"client_credentials"},
	}

	req, err := http.NewRequest(http.MethodPost, c.AccessTokenURL.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: requestTimeout}

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var authorizeResponse AuthorizeResponse

	err = json.Unmarshal(responseBody, &authorizeResponse)
	if err != nil {
		return "", err
	}

	return authorizeResponse.AccessToken, nil
}

type Store struct {
	URL *url.URL
}

func NewStore(rawURL string) (store Store, err error) {
	URL, err := url.Parse(rawURL)
	if err != nil {
		return Store{}, nil
	}

	return Store{
		URL: URL,
	}, nil
}

type UploadStatus int64

const (
	InProgress UploadStatus = iota
	Succeeded
	Failed
)

func (u UploadStatus) String() string {
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

type UploadStatusResponse struct {
	ID              string   `json:"id"`
	CreatedTime     string   `json:"createdTime"`
	LastUpdatedTime string   `json:"lastUpdatedTime"`
	Status          string   `json:"status"`
	Message         string   `json:"message"`
	ErrorCode       string   `json:"errorCode"`
	Errors          []string `json:"errors"`
}

type UpdateOptions struct {
	RetryTimeout      time.Duration
	WaitStatusTimeout time.Duration
}

func (s Store) Update(c Client, appID, filepath string, updateOptions UpdateOptions) (result UploadStatusResponse, err error) {
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
		return UploadStatusResponse{}, err
	}

	startTime := time.Now()

	for {
		if time.Now().After(startTime.Add(updateOptions.WaitStatusTimeout)) {
			return UploadStatusResponse{}, fmt.Errorf("update failed due to timeout")
		}

		log.Println("getting upload status...")

		status, err := s.UploadStatus(c, appID, string(operationID))
		if err != nil {
			return UploadStatusResponse{}, err
		}

		if status.Status == InProgress.String() {
			time.Sleep(updateOptions.RetryTimeout)

			continue
		}

		if status.Status == Succeeded.String() {
			return status, nil
		}

		if status.Status == Failed.String() {
			return UploadStatusResponse{}, fmt.Errorf("update failed due to %s, full error %+v", status.Message, status)
		}
	}
}

func (s Store) UploadUpdate(c Client, appID, filepath string) (result []byte, err error) {
	const apiPath = "/v1/products"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "submissions/draft/package")

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	req, err := http.NewRequest(http.MethodPost, apiURL, file)
	if err != nil {
		return nil, err
	}

	accessToken, err := c.Authorize()
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/zip")

	client := http.Client{Timeout: requestTimeout}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("received wrong response %s", res.Status)
	}

	operationID := res.Header.Get("Location")

	if operationID == "" {
		return nil, fmt.Errorf("received empty operation ID")
	}

	return []byte(operationID), nil
}

func (s Store) UploadStatus(c Client, appID, operationID string) (response UploadStatusResponse, err error) {
	apiPath := "v1/products"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "submissions/draft/package/operations", operationID)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return UploadStatusResponse{}, err
	}

	accessToken, err := c.Authorize()
	if err != nil {
		return UploadStatusResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := http.Client{
		Timeout: requestTimeout,
	}

	res, err := client.Do(req)
	if err != nil {
		return UploadStatusResponse{}, err
	}

	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return UploadStatusResponse{}, err
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return UploadStatusResponse{}, err
	}

	return response, nil
}
