package chrome

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

type Store struct {
	clientID     string
	clientSecret string
	refreshToken string
}

func GetStore(clientID, clientSecret, refreshToken string) Store {
	s := Store{clientID: clientID, clientSecret: clientSecret, refreshToken: refreshToken}

	return s
}

// AccessToken retrieves access token
func (s Store) AccessToken() string {
	const baseURL = "https://accounts.google.com/o/oauth2/token"
	data := url.Values{
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
		"refresh_token": {s.refreshToken},
		"grant_type":    {"refresh_token"},
		"redirect_uri":  {"urn:ietf:wg:oauth:2.0:oob"},
	}

	res, err := http.PostForm(baseURL, data)

	if err != nil {
		log.Panic(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("BODY: %s", body)

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Panic(err)
	}

	return result["access_token"].(string)
}

// Status retrieves status of the extension in the store
func (s Store) Status(appID string) string {
	accessToken := s.AccessToken()

	baseURL, err := url.Parse("https://www.googleapis.com/chromewebstore/v1.1/items/")
	if err != nil {
		log.Panic("Couldn't parse url")
	}

	baseURL.Path = path.Join(baseURL.Path, appID)

	fmt.Println("Base url: ", baseURL.String())

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		log.Panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	q := req.URL.Query()
	q.Add("projection", "DRAFT")
	req.URL.RawQuery = q.Encode()

	fmt.Println(req.URL.String())

	res, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Panic("Wasn't able to read response body")
	}

	return string(body)
}

// Insert uploads a package to create a new store item
func (s Store) Insert(body io.Reader) string {
	const baseURL = "https://www.googleapis.com/upload/chromewebstore/v1.1/items"

	accessToken := s.AccessToken()

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPost, baseURL, body)

	if err != nil {
		log.Panic(err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	result, err := client.Do(req)

	if err != nil {
		log.Panic(err)
	}

	defer result.Body.Close()

	resultBody, err := io.ReadAll(result.Body)

	if err != nil {
		log.Panic("Wasn't able to ready body response", err)
	}

	log.Println("RESULT_BODY", string(resultBody))

	return string(resultBody)
}

// Update uploads new version of the package to the store
func (s Store) Update(appID string, body io.Reader) string {
	const baseURL = "https://www.googleapis.com/upload/chromewebstore/v1.1/items/"

	accessToken := s.AccessToken()

	updateURL, err := url.Parse(baseURL)
	if err != nil {
		log.Panic("Couldn't parse url")
	}

	updateURL.Path = path.Join(updateURL.Path, appID)

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPut, updateURL.String(), body)

	if err != nil {
		log.Panic(err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	result, err := client.Do(req)

	if err != nil {
		log.Panic(err)
	}

	defer result.Body.Close()

	resultBody, err := io.ReadAll(result.Body)

	if err != nil {
		log.Panic("Wasn't able to ready body response", err)
	}

	log.Println("Update result", string(resultBody))

	return string(resultBody)
}
