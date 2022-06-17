package firefox_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maximtop/extdash/internal/firefox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err)

		assert.Equal(r.Header.Get("Authorization"), authHeader)

		_, err = w.Write([]byte(status))
		require.NoError(t, err)
	}))
	defer storeServer.Close()

	store, err := firefox.NewStore(storeServer.URL)
	require.NoError(t, err)

	actualStatus, err := store.Status(client, appID)

	require.NoError(t, err)

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
		require.NoError(t, err)

		assert.Equal(r.Header.Get("Authorization"), authHeader)
		assert.Contains(r.URL.Path, "/api/v5/addons")
		file, _, err := r.FormFile("upload")
		require.NoError(t, err)

		defer file.Close()
		body, err := io.ReadAll(file)
		require.NoError(t, err)

		assert.Contains(string(body), "test content")

		_, err = w.Write([]byte(status))
		require.NoError(t, err)
	}))
	defer storeServer.Close()

	store, err := firefox.NewStore(storeServer.URL)
	require.NoError(t, err)

	resultStatus, err := store.Insert(client, "testdata/test.txt")
	require.NoError(t, err)

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
		require.NoError(t, err)

		assert.Equal(r.Header.Get("Authorization"), authHeader)
		file, header, err := r.FormFile("upload")
		require.NoError(t, err)

		defer file.Close()
		assert.Equal(header.Filename, "extension.zip")

		_, err = w.Write([]byte(response))
		require.NoError(t, err)
	}))
	defer storeServer.Close()

	store, err := firefox.NewStore(storeServer.URL)
	require.NoError(t, err)

	actualResponse, err := store.Update(client, "testdata/extension.zip")
	require.NoError(t, err)

	assert.Equal(response, string(actualResponse))
}
