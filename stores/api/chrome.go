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
}

func GetAccessToken(clientId, clientSecret, refreshToken string) string {
	baseUrl := "https://accounts.google.com/o/oauth2/token"
	data := url.Values{
		"client_id": {clientId},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type": {"refresh_token"},
		"redirect_uri": {"urn:ietf:wg:oauth:2.0:oob"},
	}

	res, err := http.PostForm(baseUrl, data)

	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	fmt.Println("BODY: ", string(body))

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatal(err)
	}

	return result["access_token"].(string)
}

func GetStatus(appId, accessToken string) string {
	baseUrl := "https://www.googleapis.com/chromewebstore/v1.1/items/" + appId
	client := &http.Client {}
	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer " + accessToken)
	q := req.URL.Query()
	q.Add("projection", "DRAFT")
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	return string(body)
}

