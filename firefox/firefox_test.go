package firefox

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
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

	actualStatus, err := store.StatusInner(client, appID, idGenerator, 1)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(status, actualStatus)
}
