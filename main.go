package main

import (
	"fmt"
	"log"
	"net/http"
)

func root(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Extensions dashboard")
}

func main() {
	http.HandleFunc("/", root)
	http.HandleFunc("/extensions", ProcessExtensions)

	log.Fatal(http.ListenAndServe(":3000", nil))
}
