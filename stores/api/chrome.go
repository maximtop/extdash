package chrome

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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

func (s Store) GetAccessToken() string {
	baseURL := "https://accounts.google.com/o/oauth2/token"
	data := url.Values{
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
		"refresh_token": {s.refreshToken},
		"grant_type":    {"refresh_token"},
		"redirect_uri":  {"urn:ietf:wg:oauth:2.0:oob"},
	}

	res, err := http.PostForm(baseURL, data)

	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Panic(err)
	}

	fmt.Println("BODY: ", string(body))

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Panic(err)
	}

	return result["access_token"].(string)
}

func (s Store) GetStatus(appID string) string {
	accessToken := s.GetAccessToken()

	baseURL := "https://www.googleapis.com/chromewebstore/v1.1/items/" + appID
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, baseURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	q := req.URL.Query()
	q.Add("projection", "DRAFT")
	req.URL.RawQuery = q.Encode()

	fmt.Println(req.URL.String())

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Panic("wasn't able to read response body")
	}

	return string(body)
}
