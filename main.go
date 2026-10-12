package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	_ "modernc.org/sqlite"
)

type Player struct {
	Name   string
	Rating int
}

func playerExists(players []Player, name string) bool {
	for _, player := range players {
		if strings.EqualFold(player.Name, name) {
			return true
		}
	}

	return false
}

func loadPlayers(db *sql.DB) ([]Player, error) {
	rows, err := db.Query(`
		SELECT name, rating
		FROM players
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []Player

	for rows.Next() {
		var player Player

		if err := rows.Scan(&player.Name, &player.Rating); err != nil {
			return nil, err
		}

		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return players, nil
}

func main() {

	db, err := sql.Open("sqlite", "gickr.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			rating INTEGER NOT NULL DEFAULT 1000
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	players, err := loadPlayers(db)
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

		_, err := db.ExecContext(
			r.Context(),
			"INSERT INTO players (name, rating) VALUES (?, ?)",
			name,
			1000,
		)
		if err != nil {
			http.Error(w, "Could not save player", http.StatusInternalServerError)
			return
		}
		players = append(players, Player{
			Name:   name,
			Rating: 1000,
		})

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
