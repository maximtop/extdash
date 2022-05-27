package chrome

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
)

type Client struct {
	URL          string
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// Authorize retrieves access token
func (c Client) Authorize() (accessToken string, err error) {
	data := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"refresh_token": {c.RefreshToken},
		"grant_type":    {"refresh_token"},
		"redirect_uri":  {"urn:ietf:wg:oauth:2.0:oob"},
	}

	res, err := http.PostForm(c.URL, data)
	if err != nil {
		return accessToken, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return accessToken, err
	}

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return accessToken, err
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.New(result["error_description"].(string))
	}

	accessToken = result["access_token"].(string)
	return accessToken, err
}

type Store struct {
	URL string
}

type StatusResponse struct {
	Kind        string
	ID          string
	PublicKey   string
	UploadState string
	CrxVersion  string
}

// Status retrieves status of the extension in the store
func (s Store) Status(c Client, appID string) (result StatusResponse, err error) {
	const URL = "chromewebstore/v1.1/items"

	accessToken, err := c.Authorize()
	if err != nil {
		return result, err
	}

	// TODO(maximtop): !!move url parsing to the store constructor
	baseURL, err := url.Parse(s.URL)
	if err != nil {
		return result, err
	}

	baseURL.Path = path.Join(baseURL.Path, URL, appID)

	client := &http.Client{}
	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return result, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	q := req.URL.Query()
	q.Add("projection", "DRAFT")
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		err = errors.New(string(body))
		return result, err
	}

	err = json.Unmarshal(body, &result)

	if err != nil {
		return result, err
	}

	return result, err
}

type InsertResponse struct {
	Kind        string
	ID          string
	UploadState string
}

// Insert uploads a package to create a new store item
func (s Store) Insert(c Client, filePath string) (result InsertResponse, err error) {
	const URL = "upload/chromewebstore/v1.1/items"

	accessToken, err := c.Authorize()
	if err != nil {
		return
	}

	baseURL, err := url.Parse(s.URL)
	if err != nil {
		return result, err
	}
	baseURL.Path = path.Join(baseURL.Path, URL)

	body, err := os.Open(filePath)
	if err != nil {
		return result, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, baseURL.String(), body)
	if err != nil {
		return result, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	response, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}

	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return result, err
	}
	if response.StatusCode != http.StatusOK {
		return result, errors.New(string(responseBody))
	}

	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return result, err
	}

	return result, err
}

type UpdateResponse struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	UploadState string `json:"uploadState"`
}

// Update uploads new version of the package to the store
func (s Store) Update(c Client, appID, filePath string) (result UpdateResponse, err error) {
	const URL = "upload/chromewebstore/v1.1/items/"

	accessToken, err := c.Authorize()
	if err != nil {
		return result, err
	}

	updateURL, err := url.Parse(s.URL)
	if err != nil {
		return result, err
	}

	updateURL.Path = path.Join(updateURL.Path, URL, appID)

	client := &http.Client{}

	body, err := os.Open(filePath)
	if err != nil {
		return result, err
	}

	req, err := http.NewRequest(http.MethodPut, updateURL.String(), body)
	if err != nil {
		return result, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	response, err := client.Do(req)
	if err != nil {
		return result, err
	}

	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return result, err
	}

	if response.StatusCode != http.StatusOK {
		err := errors.New(string(responseBody))
		return result, err
	}

	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return result, err
	}

	return result, err
}

type PublishResponse struct {
	Kind         string   `json:"kind"`
	ItemID       string   `json:"item_id"`
	Status       []string `json:"status"`
	StatusDetail []string `json:"statusDetail"`
}

// Publish publishes app to the store
func (s Store) Publish(c Client, appID string) (result PublishResponse, err error) {
	const baseURL = "chromewebstore/v1.1/items"

	updateURL, err := url.Parse(s.URL)
	if err != nil {
		return result, err
	}

	updateURL.Path = path.Join(updateURL.Path, baseURL, appID, "publish")

	accessToken, err := c.Authorize()
	if err != nil {
		return result, err
	}

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPost, updateURL.String(), nil)
	if err != nil {
		return result, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	response, err := client.Do(req)
	if err != nil {
		return result, err
	}

	defer response.Body.Close()

	resultBody, err := io.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		err := errors.New(string(resultBody))
		return result, err
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		return result, err
	}

	if err != nil {
		return result, err
	}

	return result, err
}
