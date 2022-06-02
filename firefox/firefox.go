package firefox

import (
	"bytes"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/maximtop/extdash/helpers"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"time"
)

type Client struct {
	ClientID     string
	ClientSecret string
}

type Store struct {
	URL string
}

func genID() string {
	id := uuid.New()
	return id.String()
}

func genAuthHeader(clientID, clientSecret string, idGenerator func() string, currentTimeSec int64) (result string) {
	const expirationSec = 5 * 60

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": clientID,
		"jti": idGenerator(),
		"iat": currentTimeSec,
		"exp": currentTimeSec + expirationSec,
	})

	signedToken, err := token.SignedString([]byte(clientSecret))
	if err != nil {
		log.Panicln(err)
	}

	result = "JWT " + signedToken

	return result
}

// statusInner extracted in the separate function for testing purposes
func (s Store) statusInner(c Client, appID string, idGenerator func() string, currentTimeSec int64) (result string, err error) {
	URL := "api/v5/addons/addon/"

	baseURL, err := url.Parse(s.URL)
	if err != nil {
		return result, err
	}

	baseURL.Path = path.Join(baseURL.Path, URL, appID)
	req, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return result, err
	}

	authHeader := genAuthHeader(c.ClientID, c.ClientSecret, idGenerator, currentTimeSec)
	if err != nil {
		return result, err
	}

	req.Header.Add("Authorization", authHeader)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return result, err
	}

	if res.StatusCode != http.StatusOK {
		err = errors.New(string(body))
		return result, err
	}

	// TODO (maximtop): make identical responses for all browsers
	return string(body), err
}

// Status returns status of the extension by appID
func (s Store) Status(c Client, appID string) (result string, err error) {
	return s.statusInner(c, appID, genID, time.Now().Unix())
}

func (s Store) insertInner(
	c Client,
	filepath string,
	idGen func() string,
	currentTimeSec int64,
) (result string, err error) {
	const apiPath = "/api/v5/addons/upload"

	fullURL := helpers.JoinURL(s.URL, apiPath)

	file, err := os.Open(filepath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", file.Name())
	_, err = io.Copy(part, file)
	if err != nil {
		return result, err
	}

	req, err := http.NewRequest(http.MethodPost, fullURL, body)
	if err != nil {
		return result, err
	}
	req.Header.Add("Authorization", genAuthHeader(c.ClientID, c.ClientSecret, idGen, currentTimeSec))
	req.Header.Add("Content-Type", writer.FormDataContentType())

	dump, _ := httputil.DumpRequest(req, false)
	log.Println(string(dump))
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return result, err
	}

	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return result, err
	}

	return string(respBody), err
}

func (s Store) Insert(c Client, filename string) (result string, err error) {
	return s.insertInner(c, filename, genID, time.Now().Unix())
}
