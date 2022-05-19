package stores

import (
	chrome "extdash/stores/api"
	"github.com/gin-gonic/gin"
	"log"
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
		log.Panic("Wrong browser provided")
	}

	return chrome.Store{}
}

func ProcessStatus(ctx *gin.Context) {
	api := getAPI(ctx)

	appID := ctx.Query("app_id")
	status := api.Status(appID)

	ctx.String(http.StatusOK, status)
}

func ProcessUpdate(ctx *gin.Context) {
	ctx.String(http.StatusOK, "process update")
}

func ProcessPublish(ctx *gin.Context) {
	ctx.String(http.StatusOK, "process publish")
}

func ProcessInsert(ctx *gin.Context) {
	api := getAPI(ctx)

	api.Insert(ctx.Request.Body)
	ctx.String(http.StatusOK, "process publish")
}

// status
// update
// publish

// GET localhost:3000/extensions/status CHROME CLIENT_ID CLIENT_SECRET REFRESH_TOKEN APP_ID
