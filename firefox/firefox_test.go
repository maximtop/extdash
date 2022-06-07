package firefox

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatusInner(t *testing.T) {
	assert := assert.New(t)

	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	appID := "test_app_id"
	status := "test_status"
	currentTimeSec := time.Now().Unix()

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.Contains(r.URL.Path, appID)
		assert.Equal(r.Header.Get("Authorization"), genAuthHeader(clientID, clientSecret, currentTimeSec))

		_, err := w.Write([]byte(status))
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer storeServer.Close()

	client := Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	store := NewStore(storeServer.URL)

	actualStatus, err := store.statusInner(client, appID, currentTimeSec)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(status, string(actualStatus))
}

func TestInsertInner(t *testing.T) {
	assert := assert.New(t)

	status := "test_status"
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	currentTimeSec := time.Now().Unix()

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodPost)
		assert.Equal(r.Header.Get("Authorization"), genAuthHeader(clientID, clientSecret, currentTimeSec))
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

	client := Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	store := NewStore(storeServer.URL)

	resultStatus, err := store.insertInner(client, "testdata/test.txt", currentTimeSec)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(status, string(resultStatus))
}

func TestUpdateInner(t *testing.T) {
	assert := assert.New(t)
	response := "test_response"
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	currentTimeSec := time.Now().Unix()

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(http.MethodPut, r.Method)
		assert.Contains(r.URL.Path, "api/v5/addons/sample-for-dashboard8@adguard.com/versions/0.0.3")
		assert.Equal(r.Header.Get("Authorization"), genAuthHeader(clientID, clientSecret, currentTimeSec))
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

	client := Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	store := NewStore(storeServer.URL)

	actualResponse, err := store.updateInner(client, "testdata/extension.zip", currentTimeSec)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(response, string(actualResponse))
}
