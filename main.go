package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	_ "modernc.org/sqlite"
)

func playerExists(players []Player, name string) bool {
	for _, player := range players {
		if strings.EqualFold(player.Name, name) {
			return true
		}
	}

	return false
}

func main() {

	db, err := sql.Open("sqlite", "gickr.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := newStore(db)

	if err := store.createTables(); err != nil {
		log.Fatal(err)
	}

	players, err := store.loadPlayers()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		err := Home(players).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Could not render page", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /players", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.FormValue("name"))

		if name == "" {
			http.Error(w, "Player name is required", http.StatusBadRequest)
			return
		}

		if playerExists(players, name) {
			http.Error(w, "Player already exists", http.StatusConflict)
			return
		}

		player := Player{
			Name:   name,
			Rating: 1000,
		}

		if err := store.addPlayer(r.Context(), player); err != nil {
			http.Error(w, "Could not save player", http.StatusInternalServerError)
			return
		}

		players = append(players, player)

		if r.Header.Get("HX-Request") == "true" {
			err := PlayerList(players).Render(r.Context(), w)
			if err != nil {
				http.Error(w, "Could not render player list", http.StatusInternalServerError)
			}
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	log.Fatal(http.ListenAndServe(":8080", mux))

}
