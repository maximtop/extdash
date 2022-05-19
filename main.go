package main

import (
	"extdash/stores"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

func root(c *gin.Context) {
	c.String(http.StatusOK, "Extensions dashboard")
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	router := gin.Default()

	router.GET("/", root)

	router.GET("/stores/:browser", stores.ProcessStatus)
	router.POST("/stores/:browser", stores.ProcessInsert)
	router.PUT("/stores/:browser", stores.ProcessUpdate)
	router.POST("/stores/:browser/publish", stores.ProcessPublish)

	log.Fatal(router.Run(":" + port))
}
