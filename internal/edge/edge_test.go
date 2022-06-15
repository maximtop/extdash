package edge_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/maximtop/extdash/internal/edge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuthServer(t *testing.T, accessToken string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := edge.AuthorizeResponse{
			TokenType:   "",
			ExpiresIn:   0,
			AccessToken: accessToken,
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		_, err = w.Write(responseData)
		if err != nil {
			t.Fatal(err)
		}
	}))
}

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

func TestUploadUpdate(t *testing.T) {
	assert := assert.New(t)
	accessToken := "test_access_token"
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	appID := "test_app_id"
	operationID := "test_operation_id"

	authServer := newAuthServer(t, accessToken)

	client, err := edge.NewClient(clientID, clientSecret, authServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(http.MethodPost, r.Method)
		assert.Equal("Bearer "+accessToken, r.Header.Get("Authorization"))
		assert.Equal("application/zip", r.Header.Get("Content-Type"))
		assert.Equal(path.Join("/v1/products", appID, "submissions/draft/package"), r.URL.Path)

		responseBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal("test_file_content", string(responseBody))

		w.Header().Set("Location", operationID)
		w.WriteHeader(http.StatusAccepted)

		_, err = w.Write(nil)
		if err != nil {
			t.Fatal(err)
		}
	}))

	store, err := edge.NewStore(storeServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	actualUpdateResponse, err := store.UploadUpdate(client, appID, "./testdata/test.txt")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(operationID, string(actualUpdateResponse))
}

func TestUploadStatus(t *testing.T) {
	assert := assert.New(t)
	accessToken := "test_access_token"
	response := edge.UploadStatusResponse{
		ID:              "{operationID}",
		CreatedTime:     "Date Time",
		LastUpdatedTime: "Date Time",
		Status:          "Failed",
		Message:         "Error Message.",
		ErrorCode:       "Error Code",
		Errors:          []string{"list of errors"},
	}
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	appID := "test_app_id"
	operationID := "test_operation_id"

	authServer := newAuthServer(t, accessToken)

	client, err := edge.NewClient(clientID, clientSecret, authServer.URL)
	require.NoError(t, err)

	storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Header.Get("Authorization"), "Bearer "+accessToken)
		assert.Equal(r.URL.Path, "/v1/products/"+appID+"/submissions/draft/package/operations/"+operationID)

		response, err := json.Marshal(response)
		require.NoError(t, err)

		_, err = w.Write(response)
		require.NoError(t, err)
	}))

	store, err := edge.NewStore(storeServer.URL)
	require.NoError(t, err)

	uploadStatus, err := store.UploadStatus(client, appID, operationID)
	require.NoError(t, err)

	assert.Equal(response, uploadStatus)
}

func TestUpdate(t *testing.T) {
	clientID := "test_client_id"
	clientSecret := "test_client_secret"
	accessToken := "test_access_token"
	appID := "test_app_id"
	operationID := "test_operation_id"
	filepath := "testdata/test.txt"

	t.Run("waits for successful response", func(t *testing.T) {
		succeededResponse := edge.UploadStatusResponse{
			ID:              "",
			CreatedTime:     "",
			LastUpdatedTime: "",
			Status:          edge.Succeeded.String(),
			Message:         "",
			ErrorCode:       "",
			Errors:          nil,
		}

		authServer := newAuthServer(t, accessToken)
		client, err := edge.NewClient(clientID, clientSecret, authServer.URL)
		if err != nil {
			t.Fatal(err)
		}

		counter := 0
		storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "submissions/draft/package/operations") {
				if counter == 0 {
					inProgressResponse, err := json.Marshal(edge.UploadStatusResponse{
						ID:              "",
						CreatedTime:     "",
						LastUpdatedTime: "",
						Status:          edge.InProgress.String(),
						Message:         "",
						ErrorCode:       "",
						Errors:          nil,
					})
					require.NoError(t, err)

					_, err = w.Write(inProgressResponse)
					require.NoError(t, err)
				}
				if counter == 1 {
					marshaledSucceededResponse, err := json.Marshal(succeededResponse)
					require.NoError(t, err)

					_, err = w.Write(marshaledSucceededResponse)
					require.NoError(t, err)
				}
				counter++

				return
			}

			w.Header().Set("Location", operationID)
			w.WriteHeader(http.StatusAccepted)

			_, err := w.Write(nil)
			require.NoError(t, err)
		}))
		defer storeServer.Close() // FIXME check that all servers are closed

		store, err := edge.NewStore(storeServer.URL)
		if err != nil {
			t.Fatal(err)
		}

		response, err := store.Update(
			client,
			appID,
			filepath,
			edge.UpdateOptions{
				RetryTimeout: time.Nanosecond,
			})
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, succeededResponse, response)
	})

	t.Run("throws error on timeout", func(t *testing.T) {
		updateOptions := edge.UpdateOptions{
			RetryTimeout:      time.Millisecond,
			WaitStatusTimeout: 2 * time.Millisecond,
		}

		authServer := newAuthServer(t, accessToken)
		client, err := edge.NewClient(clientID, clientSecret, authServer.URL)
		require.NoError(t, err)

		storeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "submissions/draft/package/operations") {
				inProgressResponse, err := json.Marshal(edge.UploadStatusResponse{
					ID:              "",
					CreatedTime:     "",
					LastUpdatedTime: "",
					Status:          edge.InProgress.String(),
					Message:         "",
					ErrorCode:       "",
					Errors:          nil,
				})
				require.NoError(t, err)

				_, err = w.Write(inProgressResponse)
				require.NoError(t, err)
				return
			}

			w.Header().Set("Location", operationID)
			w.WriteHeader(http.StatusAccepted)

			_, err := w.Write(nil)
			require.NoError(t, err)
		}))

		store, err := edge.NewStore(storeServer.URL)
		require.NoError(t, err)

		_, err = store.Update(client, appID, filepath, updateOptions)
		log.Println(err)
		assert.ErrorContains(t, err, "update failed due to timeout")
	})
}
