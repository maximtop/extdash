package stores

import (
	chrome "extdash/stores/api"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Browser string

const (
	Chrome Browser = "chrome"
	//Firefox string = "firefox"
	//Edge string = "edge"
	//Opera string = "opera"
)

type StoreApi interface {
	getStatus(ctx *gin.Context) string
}

func getChromeStatus(c *gin.Context) string {
	// FIXME мы будем эти параметры хранить в переменных окружения или передавать снаружи каждый раз?
	clientId := c.Query("client_id")
	clientSecret := c.Query("client_secret")
	refreshToken := c.Query("refresh_token")
	appId := c.Query("app_id")

	accessToken := chrome.GetAccessToken(clientId, clientSecret, refreshToken)

	chrome.GetStatus(appId, accessToken)

	return "ok"
}

func getApi(browser Browser) StoreApi {
	switch browser {
	case Chrome:
		return chrome.Chrome
	default:
		panic("Wrong browser provided")
	}
}

func ProcessStatus(ctx *gin.Context) {
	browser := Browser(ctx.Param("browser"))

	api := getApi(browser)

	api.getStatus(ctx)
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