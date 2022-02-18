package main

import (
	"fmt"
	"net/http"
)

func ProcessExtensions(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "process extensions")
}
