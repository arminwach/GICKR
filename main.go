package main

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
)

type Player struct {
	Name   string
	Rating int
}

func main() {

	players := []Player{
		{Name: "Armin", Rating: 1000},
		{Name: "Friend", Rating: 1000},
	}

	mux := http.NewServeMux()

	mux.Handle("GET /", templ.Handler(Home(players)))

	log.Fatal(http.ListenAndServe(":8080", mux))

}
