package stores

import (
	chrome "extdash/stores/api"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Browser string

const (
	Chrome Browser = "chrome"
	// Firefox string = "firefox"
	// Edge string = "edge"
	// Opera string = "opera"
)

func getAPI(ctx *gin.Context) chrome.Store {
	browser := Browser(ctx.Param("browser"))

	clientID := ctx.Query("client_id")
	clientSecret := ctx.Query("client_secret")
	refreshToken := ctx.Query("refresh_token")

	switch browser {
	case Chrome:
		return chrome.GetStore(clientID, clientSecret, refreshToken)
	default:
		panic("Wrong browser provided")
	}
}

func ProcessStatus(ctx *gin.Context) {
	api := getAPI(ctx)

	appID := ctx.Query("app_id")
	status := api.GetStatus(appID)

	ctx.String(http.StatusOK, status)
}

func ProcessUpdate(c *gin.Context) {
	c.String(http.StatusOK, "process update")
}

func ProcessPublish(c *gin.Context) {
	c.String(http.StatusOK, "process publish")
}

// status
// update
// publish

// GET localhost:3000/extensions/status CHROME CLIENT_ID CLIENT_SECRET REFRESH_TOKEN APP_ID
