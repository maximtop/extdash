package firefox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/maximtop/extdash/fileutil"
	"github.com/maximtop/extdash/urlutil"
)

// TODO add method for signing standalone extension

// Client describes client structure
type Client struct {
	clientID     string
	clientSecret string
	now          func() int64
}

type ClientConfig struct {
	ClientID     string
	ClientSecret string
	Now          func() int64
}

const requestTimeout = 20 * time.Minute

// NewClient creates instance of the Client
func NewClient(config ClientConfig) Client {
	c := config

	if c.Now == nil {
		c.Now = func() int64 {
			return time.Now().Unix()
		}
	}

	return Client{
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		now:          c.Now,
	}
}

// GenAuthHeader generates header used for authorization
func (c Client) GenAuthHeader() (result string, err error) {
	const expirationSec = 5 * 60

	currentTimeSec := c.now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": c.clientID,
		"iat": currentTimeSec,
		"exp": currentTimeSec + expirationSec,
	})

	signedToken, err := token.SignedString([]byte(c.clientSecret))
	if err != nil {
		return "", err
	}

	return "JWT " + signedToken, nil
}

// Store type describes store structure
type Store struct {
	URL *url.URL
}

// NewStore parses rawUrl and creates instance of the Store
func NewStore(rawURL string) (s Store, err error) {
	URL, err := url.Parse(rawURL)
	if err != nil {
		return Store{}, fmt.Errorf("wasn't able to parse url %s due to: %w", rawURL, err)
	}
	return Store{URL: URL}, nil
}

// Manifest describes required fields parsed from the manifest
type Manifest struct {
	Version      string
	Applications struct {
		Gecko struct {
			ID string
		}
	}
}

// parseManifest reads zip archive, and extracts manifest.json out of it
func parseManifest(zipFilepath string) (result Manifest, err error) {
	fileContent, err := fileutil.ReadFileFromZip(zipFilepath, "manifest.json")
	if err != nil {
		return Manifest{}, err
	}

	err = json.Unmarshal(fileContent, &result)
	if err != nil {
		return Manifest{}, err
	}
	return result, nil
}

// Status returns status of the extension by appID
func (s *Store) Status(c Client, appID string) (result []byte, err error) {
	apiPath := "api/v5/addons/addon/"

	apiURL := urlutil.JoinURL(s.URL, apiPath, appID)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	authHeader, err := c.GenAuthHeader()
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", authHeader)

	client := &http.Client{Timeout: requestTimeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, body)
	}

	// TODO (maximtop): make identical responses for all browsers
	return body, nil
}

// Insert uploads extension to the amo
// https://addons-server.readthedocs.io/en/latest/topics/api/signing.html?highlight=%2Faddons%2F#post--api-v5-addons-
// CURL example:
// curl -v -XPOST \
//  -H "Authorization: JWT ${ACCESS_TOKEN}" \
//  -F "upload=@tmp/extension.zip" \
//  "https://addons.mozilla.org/api/v5/addons/"
func (s *Store) Insert(c Client, filepath string) (result []byte, err error) {
	const apiPath = "/api/v5/addons/"

	// trailing slash is required for this request
	// in go 1.19 would be possible u.JoinPath("users", "/")
	apiURL := urlutil.JoinURL(s.URL, apiPath) + "/"

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("upload", path.Base(file.Name()))
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, body)
	if err != nil {
		return nil, err
	}

	authHeader, err := c.GenAuthHeader()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	client := http.Client{Timeout: requestTimeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

// Update uploads new version of extension to the store
// Before uploading it reads manifest.json for getting extension version and uuid
func (s *Store) Update(c Client, filepath string) (result []byte, err error) {
	const apiPath = "api/v5/addons"

	manifest, err := parseManifest(filepath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	apiURL := urlutil.JoinURL(s.URL, apiPath, manifest.Applications.Gecko.ID, "versions", manifest.Version) + "/"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("upload", path.Base(file.Name()))
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	client := http.Client{Timeout: requestTimeout}
	req, err := http.NewRequest(http.MethodPut, apiURL, body)
	if err != nil {
		return nil, err
	}

	authHeader, err := c.GenAuthHeader()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return responseBody, nil
}
