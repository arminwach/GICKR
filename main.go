package main

import (
	"log"
	"net/http"
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

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		err := Home(players).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Could not render page", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /players", func(w http.ResponseWriter, r *http.Request) {
		name := r.FormValue("name")

		players = append(players, Player{
			Name:   name,
			Rating: 1000,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	log.Fatal(http.ListenAndServe(":8080", mux))

}
