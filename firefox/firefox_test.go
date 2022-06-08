package firefox_test

import (
	"github.com/maximtop/extdash/firefox"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
	assert := assert.New(t)

	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	appID := "test_app_id"
	status := "test_status"
	now := func() int64 {
		return 1
	}

	client := firefox.NewClient(firefox.ClientConfig{ClientID: clientID, ClientSecret: clientSecret, Now: now})

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.Contains(r.URL.Path, appID)
		authHeader, err := client.GenAuthHeader()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(r.Header.Get("Authorization"), authHeader)

		_, err = w.Write([]byte(status))
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer storeServer.Close()

	store, err := firefox.NewStore(storeServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	actualStatus, err := store.Status(client, appID)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(status, string(actualStatus))
}

func TestInsert(t *testing.T) {
	assert := assert.New(t)

	status := "test_status"
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	currentTimeSec := time.Now().Unix()
	now := func() int64 {
		return currentTimeSec
	}

	client := firefox.NewClient(firefox.ClientConfig{ClientID: clientID, ClientSecret: clientSecret, Now: now})

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodPost)
		authHeader, err := client.GenAuthHeader()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(r.Header.Get("Authorization"), authHeader)
		assert.Contains(r.URL.Path, "/api/v5/addons")
		file, _, err := r.FormFile("upload")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatal(err)
		}
		assert.Contains(string(body), "test content")

		_, err = w.Write([]byte(status))
		if err != nil {
			t.Fatal(err)
		}
	}))

	store, err := firefox.NewStore(storeServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	resultStatus, err := store.Insert(client, "testdata/test.txt")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(status, string(resultStatus))
}

func TestUpdate(t *testing.T) {
	assert := assert.New(t)
	response := "test_response"
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	currentTimeSec := time.Now().Unix()
	now := func() int64 {
		return currentTimeSec
	}

	client := firefox.NewClient(firefox.ClientConfig{ClientID: clientID, ClientSecret: clientSecret, Now: now})

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(http.MethodPut, r.Method)
		assert.Contains(r.URL.Path, "api/v5/addons/sample-for-dashboard8@adguard.com/versions/0.0.3")
		authHeader, err := client.GenAuthHeader()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(r.Header.Get("Authorization"), authHeader)
		file, header, err := r.FormFile("upload")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		assert.Equal(header.Filename, "extension.zip")

		_, err = w.Write([]byte(response))
		if err != nil {
			t.Fatal(err)
		}
	}))

	store, err := firefox.NewStore(storeServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	actualResponse, err := store.Update(client, "testdata/extension.zip")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(response, string(actualResponse))
}
