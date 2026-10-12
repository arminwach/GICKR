package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
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

func renderPlayerFormError(w http.ResponseWriter, r *http.Request, message string) {
	if r.Header.Get("HX-Request") != "true" {
		http.Error(w, message, http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("HX-Retarget", "#player-form-error")
	w.Header().Set("HX-Reswap", "innerHTML")
	if err := PlayerFormError(message).Render(r.Context(), w); err != nil {
		log.Printf("render player form error: %v", err)
	}
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
			renderPlayerFormError(w, r, "Player name is required")
			return
		}

		if playerExists(players, name) {
			renderPlayerFormError(w, r, "Player already exists")
			return
		}

		player := Player{
			Name:   name,
			Rating: 1000,
		}

		player, err = store.addPlayer(r.Context(), player)
		if err != nil {
			http.Error(w, "Could not save player", http.StatusInternalServerError)
			return
		}

		players = append(players, player)

		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("X-Player-Added", "true")
			err := PlayerList(players).Render(r.Context(), w)
			if err != nil {
				http.Error(w, "Could not render player list", http.StatusInternalServerError)
			}
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("DELETE /players/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid player ID", http.StatusBadRequest)
			return
		}

		if err := store.deletePlayer(r.Context(), id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Player not found", http.StatusNotFound)
				return
			}

			http.Error(w, "Could not delete player", http.StatusInternalServerError)
			return
		}

		players, err = store.loadPlayers()
		if err != nil {
			http.Error(w, "Could not load players", http.StatusInternalServerError)
			return
		}

		if err := PlayerList(players).Render(r.Context(), w); err != nil {
			http.Error(w, "Could not render player list", http.StatusInternalServerError)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", mux))

}
