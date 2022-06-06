package firefox

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/maximtop/extdash/urlutil"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

// TODO add method for signing standalone extension

type Client struct {
	ClientID     string
	ClientSecret string
}

type Store struct {
	URL string
}

func genAuthHeader(clientID, clientSecret string, currentTimeSec int64) (result string) {
	const expirationSec = 5 * 60

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": clientID,
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
func (s Store) statusInner(c Client, appID string, currentTimeSec int64) (result string, err error) {
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

	authHeader := genAuthHeader(c.ClientID, c.ClientSecret, currentTimeSec)
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
	return s.statusInner(c, appID, time.Now().Unix())
}

// insertInner extracted in the separate method for testing purposes
// https://addons-server.readthedocs.io/en/latest/topics/api/signing.html?highlight=%2Faddons%2F#post--api-v5-addons-
// CURL example:
// curl -v -XPOST \
//  -H "Authorization: JWT ${ACCESS_TOKEN}" \
//  -F "upload=@tmp/extension.zip" \
//  "https://addons.mozilla.org/api/v5/addons/"
func (s Store) insertInner(
	c Client,
	filepath string,
	currentTimeSec int64,
) (result string, err error) {
	const apiPath = "/api/v5/addons/"

	// trailing slash is required for this request
	// in go 1.19 would be possible u.JoinPath("users", "/")
	fullURL := urlutil.JoinURL(s.URL, apiPath) + "/"

	file, err := os.Open(filepath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("upload", path.Base(file.Name()))
	_, err = io.Copy(part, file)
	if err != nil {
		return result, err
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, fullURL+"/", body)
	if err != nil {
		return result, err
	}
	req.Header.Add("Authorization", genAuthHeader(c.ClientID, c.ClientSecret, currentTimeSec))
	req.Header.Add("Content-Type", writer.FormDataContentType())

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

type Manifest struct {
	Version      string
	Applications struct {
		Gecko struct {
			ID string
		}
	}
}

func readFile(file *zip.File) (result []byte, err error) {
	reader, err := file.Open()
	if err != nil {
		return result, err
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return result, err
	}
	return content, err
}

func parseManifest(filepath string) (result Manifest, err error) {
	reader, err := zip.OpenReader(filepath)
	if err != nil {
		return result, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == "manifest.json" {
			fileContent, err := readFile(file)
			if err != nil {
				return result, err
			}

			err = json.Unmarshal(fileContent, &result)
			if err != nil {
				return result, err
			}
			return result, err
		}
	}

	return Manifest{}, errors.New("Wasn't able to find manifest file in the" + filepath)
}

// Insert uploads extension to the amo
func (s Store) Insert(c Client, filepath string) (result string, err error) {
	return s.insertInner(c, filepath, time.Now().Unix())
}

func (s Store) updateInner(c Client, filepath string, currentTimeSec int64) (result string, err error) {
	const apiPath = "api/v5/addons"

	manifest, err := parseManifest(filepath)
	if err != nil {
		return result, err
	}

	file, err := os.Open(filepath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	baseURL, err := url.Parse(s.URL)
	if err != nil {
		return result, err
	}

	baseURL.Path = path.Join(baseURL.Path, apiPath, manifest.Applications.Gecko.ID, "versions", manifest.Version)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("upload", path.Base(file.Name()))
	_, err = io.Copy(part, file)
	if err != nil {
		return result, err
	}
	writer.Close()

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, baseURL.String()+"/", body)
	if err != nil {
		return result, err
	}

	authHeader := genAuthHeader(c.ClientID, c.ClientSecret, currentTimeSec)
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return result, err
	}

	return string(responseBody), nil
}

func (s Store) Update(c Client, filepath string) (result string, err error) {
	return s.updateInner(c, filepath, time.Now().Unix())
}
