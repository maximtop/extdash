package firefox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/maximtop/extdash/internal/fileutil"
	"github.com/maximtop/extdash/internal/urlutil"
)

// AMO main url is https://addons.mozilla.org/
// Please use AMO dev environment at https://addons-dev.allizom.org/ or the AMO stage
// environment at https://addons.allizom.org/ for testing.
// credential keys can't be sent via email, so you need to ask them in the chat https://matrix.to/#/#amo:mozilla.org
// last time I've asked them from mat https://matrix.to/#/@mat:mozilla.org
// signed xpi build from dev environments is corrupted, so you need to build it from the production environment

// Client describes client structure.
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

// TODO(maximtop): consider to make this constant an option.
const requestTimeout = 20 * time.Minute

// maxReadLimit limits response size returned from the store api.
const maxReadLimit = 10 * fileutil.MB

// NewClient creates instance of the Client.
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

// GenAuthHeader generates header used for authorization.
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

// Store type describes store structure.
type Store struct {
	URL *url.URL
}

// NewStore parses rawUrl and creates instance of the Store.
func NewStore(rawURL string) (s Store, err error) {
	URL, err := url.Parse(rawURL)
	if err != nil {
		return Store{}, fmt.Errorf("wasn't able to parse url %s due to: %w", rawURL, err)
	}

	return Store{URL: URL}, nil
}

// Manifest describes required fields parsed from the manifest.
type Manifest struct {
	Version      string
	Applications struct {
		Gecko struct {
			ID string
		}
	}
}

// parseManifest reads zip archive, and extracts manifest.json out of it.
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

// Status returns status of the extension by appID.
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

	body, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, body)
	}

	// TODO(maximtop): make identical responses for all browsers
	return body, nil
}

type Version struct {
	ID      int    `json:"id"`
	Version string `json:"version"`
}

type VersionResponse struct {
	PageSize  int         `json:"page_size"`
	PageCount int         `json:"page_count"`
	Count     int         `json:"count"`
	Next      interface{} `json:"next"`
	Previous  interface{} `json:"previous"`
	Results   []Version   `json:"results"`
}

// VersionID retrieves version ID by version number.
func (s *Store) VersionID(c Client, appID, version string) (result string, err error) {
	log.Printf("[DEBUG] Getting version ID for appID: %s, version: %s", appID, version)

	const apiPath = "api/v5/addons/addon/"

	queryString := url.Values{}
	queryString.Add("filter", "all_with_unlisted")
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "versions") + "?" + queryString.Encode()

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	authHeader, err := c.GenAuthHeader()
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", authHeader)

	client := &http.Client{Timeout: requestTimeout}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got code %d, body: %q", res.StatusCode, body)
	}

	var versions VersionResponse

	err = json.Unmarshal(body, &versions)
	if err != nil {
		return "", err
	}

	var versionID string

	for _, resultVersion := range versions.Results {
		if resultVersion.Version == version {
			versionID = strconv.Itoa(resultVersion.ID)
			break
		}
	}

	if versionID == "" {
		return "", fmt.Errorf("version %s not found", version)
	}

	log.Printf("[DEBUG] Version ID: %s", versionID)

	return versionID, nil
}

// UploadSource uploads source code of the extension to the store.
// Source can be uploaded only after the extension is validated.
func (s *Store) UploadSource(c Client, appID, versionID, sourcePath string) (result []byte, err error) {
	log.Printf("[DEBUG] Uploading source for appID: %s, versionID: %s", appID, versionID)

	const apiPath = "api/v5/addons/addon/"

	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "versions", versionID) + "/"

	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("source", file.Name())
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

	req, err := http.NewRequest(http.MethodPatch, apiURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

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

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, body)
	}

	log.Println("[DEBUG] Successfully uploaded source")

	return responseBody, nil
}

type UploadStatusFiles struct {
	DownloadURL string `json:"download_url"`
	Hash        string `json:"hash"`
	Signed      bool   `json:"signed"`
}

type UploadStatus struct {
	GUID             string              `json:"guid"`
	Active           bool                `json:"active"`
	AutomatedSigning bool                `json:"automated_signing"`
	Files            []UploadStatusFiles `json:"files"`
	PassedReview     bool                `json:"passed_review"`
	Pk               string              `json:"pk"`
	Processed        bool                `json:"processed"`
	Reviewed         ReviewedStatus      `json:"reviewed"`
	URL              string              `json:"url"`
	Valid            bool                `json:"valid"`
	ValidationURL    string              `json:"validation_url"`
	Version          string              `json:"version"`
}

type ReviewedStatus bool

// UnmarshalJSON parses ReviewedStatus.
// used because review status may be boolean or string
func (w *ReviewedStatus) UnmarshalJSON(b []byte) error {
	stringVal := string(b)

	boolVal, err := strconv.ParseBool(stringVal)
	if err == nil {
		*w = ReviewedStatus(boolVal)
		return nil
	}
	if len(stringVal) > 0 {
		*w = true
	} else {
		*w = false
	}

	return nil
}

// UploadStatus retrieves upload status of the extension.
// curl "https://addons.mozilla.org/api/v5/addons/@my-addon/versions/1.0/"
//    -g -H "Authorization: JWT <jwt-token>"
func (s *Store) UploadStatus(c Client, appID, version string) (status *UploadStatus, err error) {
	log.Printf("[DEBUG] Getting upload status for appID: %s, version: %s", appID, version)

	const apiPath = "api/v5/addons"
	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "versions", version)

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

	body, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, body)
	}

	var uploadStatus UploadStatus

	err = json.Unmarshal(body, &uploadStatus)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Received upload status: %+v", uploadStatus)

	return &uploadStatus, nil
}

// AwaitValidation awaits validation of the extension.
func (s *Store) AwaitValidation(c Client, appID, version string) (err error) {
	// TODO(maximtop): move constants to config
	const retryInterval = time.Second
	const maxAwaitTime = time.Minute * 20

	var startTime = time.Now()

	for {
		if (time.Now().Sub(startTime)) > maxAwaitTime {
			return fmt.Errorf("await validation timeout")
		}

		uploadStatus, err := s.UploadStatus(c, appID, version)
		if err != nil {
			return err
		}
		if uploadStatus.Processed {
			log.Println("[DEBUG] Extension upload processed successfully")
			break
		} else {
			time.Sleep(retryInterval)
		}
	}

	return nil
}

// UploadNew uploads the extension to the store for the first time
// https://addons-server.readthedocs.io/en/latest/topics/api/signing.html?highlight=%2Faddons%2F#post--api-v5-addons-
// CURL example:
// curl -v -XPOST \
//  -H "Authorization: JWT ${ACCESS_TOKEN}" \
//  -F "upload=@tmp/extension.zip" \
//  "https://addons.mozilla.org/api/v5/addons/"
func (s *Store) UploadNew(c Client, filepath string) (result []byte, err error) {
	log.Printf("[DEBUG] Uploading new extension: %s", filepath)

	const apiPath = "api/v5/addons"

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

	part, err := writer.CreateFormFile("upload", file.Name())
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

	respBody, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, respBody)
	}

	log.Printf("[DEBUG] Uploaded new extension: %s, response: %s", filepath, respBody)

	return respBody, nil
}

// Insert uploads extension to the amo for the first time.
func (s *Store) Insert(c Client, filepath, sourcepath string) (err error) {
	log.Printf("[DEBUG] Start uploading new extension: %s, with source: %s", filepath, sourcepath)

	_, err = s.UploadNew(c, filepath)
	if err != nil {
		return err
	}

	manifest, err := parseManifest(filepath)
	if err != nil {
		return err
	}

	appID := manifest.Applications.Gecko.ID
	version := manifest.Version

	err = s.AwaitValidation(c, appID, version)
	if err != nil {
		return err
	}

	versionID, err := s.VersionID(c, appID, version)
	if err != nil {
		return err
	}

	_, err = s.UploadSource(c, appID, versionID, sourcepath)
	if err != nil {
		return err
	}

	return nil
}

// UploadUpdate uploads the extension update.
func (s *Store) UploadUpdate(c Client, appID, version, filepath string) (result []byte, err error) {
	log.Printf("[DEBUG] Start uploading update for extension: %s", filepath)

	const apiPath = "api/v5/addons"

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	apiURL := urlutil.JoinURL(s.URL, apiPath, appID, "versions", version) + "/"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("upload", file.Name())
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

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("got code %d, body: %q", res.StatusCode, responseBody)
	}

	log.Printf("[DEBUG] Successfully uploaded update for extension: %s, response: %s", filepath, responseBody)

	return responseBody, nil
}

// Update uploads new version of extension to the store
// Before uploading it reads manifest.json for getting extension version and uuid.
func (s *Store) Update(c Client, filepath, sourcepath string) (err error) {
	log.Printf("[DEBUG] Start uploading update for extension: %s, with source: %s", filepath, sourcepath)

	manifest, err := parseManifest(filepath)
	if err != nil {
		return err
	}

	appID := manifest.Applications.Gecko.ID
	version := manifest.Version

	_, err = s.UploadUpdate(c, appID, version, filepath)
	if err != nil {
		return err
	}

	err = s.AwaitValidation(c, appID, version)
	if err != nil {
		return err
	}

	versionID, err := s.VersionID(c, appID, version)
	if err != nil {
		return err
	}

	_, err = s.UploadSource(c, appID, versionID, sourcepath)
	if err != nil {
		return err
	}

	return nil
}

// AwaitSigning waits for the extension to be signed.
func (s *Store) AwaitSigning(c Client, appID, version string) (err error) {
	log.Printf("[DEBUG] Start waiting for signing of extension: %s", appID)

	// TODO(maximtop): move constants to config
	const retryInterval = time.Second
	const maxAwaitTime = time.Minute * 20

	var startTime = time.Now()

	for {
		if (time.Now().Sub(startTime)) > maxAwaitTime {
			return fmt.Errorf("await signing timeout")
		}

		uploadStatus, err := s.UploadStatus(c, appID, version)
		if err != nil {
			return err
		}

		var signedAndReady = uploadStatus.Valid && uploadStatus.Active && bool(uploadStatus.Reviewed) && len(uploadStatus.Files) > 0
		var requiresManualReview = uploadStatus.Valid && !uploadStatus.AutomatedSigning

		if signedAndReady || requiresManualReview {
			if requiresManualReview {
				return fmt.Errorf("extension won't be signed automatically, status: %+v", uploadStatus)
			}
			if signedAndReady {
				log.Printf("[DEBUG] Extension is signed and ready: %s", appID)
				return nil
			}
		} else {
			time.Sleep(retryInterval)
		}
	}

	return nil
}

// DownloadSigned downloads signed extension.
func (s *Store) DownloadSigned(c Client, appID, version string) (err error) {
	log.Printf("[DEBUG] Start downloading signed extension: %s", appID)

	uploadStatus, err := s.UploadStatus(c, appID, version)
	if err != nil {
		return err
	}

	if len(uploadStatus.Files) == 0 {
		return fmt.Errorf("no files to download")
	}

	var downloadURL = uploadStatus.Files[0].DownloadURL

	client := http.Client{Timeout: requestTimeout}

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}

	authHeader, err := c.GenAuthHeader()
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", authHeader)

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxReadLimit))
	if err != nil {
		return err
	}

	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return err
	}

	var filename = path.Base(parsedURL.Path)

	// save response to file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(file, bytes.NewReader(responseBody))
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}

// Sign uploads the extension to the store, waits for signing, downloads and saves the signed
// extension in the directory
func (s *Store) Sign(c Client, filepath string) (err error) {
	log.Printf("[DEBUG] Start signing extension: %s", filepath)

	manifest, err := parseManifest(filepath)
	if err != nil {
		return err
	}

	appID := manifest.Applications.Gecko.ID
	version := manifest.Version

	_, err = s.UploadUpdate(c, appID, version, filepath)
	if err != nil {
		return err
	}

	err = s.AwaitSigning(c, appID, version)
	if err != nil {
		return err
	}

	err = s.DownloadSigned(c, appID, version)
	if err != nil {
		return err
	}

	return nil
}
