package edge

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const requestTimeout = 5 * time.Second

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

func (c Client) Authorize() (accessToken string, err error) {
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

	json.Unmarshal(responseBody, &authorizeResponse)

	return authorizeResponse.AccessToken, nil
}
