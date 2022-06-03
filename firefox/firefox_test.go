package firefox

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatus(t *testing.T) {
	assert := assert.New(t)

	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	appID := "test_app_id"
	status := "test_status"

	idGenerator := func() string {
		return "test_id"
	}

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.Contains(r.URL.Path, appID)
		assert.Equal(r.Header.Get("Authorization"), genAuthHeader(clientID, clientSecret, idGenerator, 1))

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

	store := Store{URL: storeServer.URL}

	actualStatus, err := store.statusInner(client, appID, idGenerator, 1)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(status, actualStatus)
}

func TestInsertInner(t *testing.T) {
	assert := assert.New(t)

	status := "test_status"
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	idGen := func() string {
		return "test_id"
	}
	currentTimeSec := time.Now().Unix()

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodPost)
		assert.Equal(r.Header.Get("Authorization"), genAuthHeader(clientID, clientSecret, idGen, currentTimeSec))
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

	store := Store{
		URL: storeServer.URL,
	}

	resultStatus, err := store.insertInner(client, "testdata/test.txt", idGen, currentTimeSec)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(status, resultStatus)
}
