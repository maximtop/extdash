package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func root(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Extensions dashboard")
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	http.HandleFunc("/", root)
	http.HandleFunc("/extensions", ProcessExtensions)

	log.Fatal(http.ListenAndServe(":" + port, nil))
}
