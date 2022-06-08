package edge_test

import (
	"encoding/json"
	"github.com/maximtop/extdash/edge"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorize(t *testing.T) {
	assert := assert.New(t)

	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	accessToken := "test_access_token"

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(req.Method, http.MethodPost)
		assert.Equal(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		assert.Equal(req.FormValue("client_id"), clientID)
		assert.Equal(req.FormValue("scope"), "https://api.addons.microsoftedge.microsoft.com/.default")
		assert.Equal(req.FormValue("client_secret"), clientSecret)
		assert.Equal(req.FormValue("grant_type"), "client_credentials")

		response, err := json.Marshal(edge.AuthorizeResponse{
			TokenType:   "",
			ExpiresIn:   0,
			AccessToken: accessToken,
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = w.Write(response)
		if err != nil {
			t.Fatal(err)
		}
	}))

	client, err := edge.NewClient(clientID, clientSecret, authServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	actualAccessToken, err := client.Authorize()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(accessToken, actualAccessToken)
}
