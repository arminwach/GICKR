package main

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
)

func main() {

	mux := http.NewServeMux()

	mux.Handle("GET /", templ.Handler(Home()))

	log.Fatal(http.ListenAndServe(":8080", mux))

}
