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
	router.POST("/stores/:browser/publish", stores.ProcessPublish)
	router.PATCH("/stores/:browser", stores.ProcessUpdate)

	log.Fatal(router.Run(":" + port))
}
