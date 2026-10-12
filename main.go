package main

import (
	"database/sql"
	"errors"
	"fmt"
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

func playerFromForm(r *http.Request, players []Player, field string, required bool) (Player, bool, error) {
	value := strings.TrimSpace(r.FormValue(field))
	if value == "" {
		if required {
			return Player{}, false, fmt.Errorf("select every required player")
		}
		return Player{}, false, nil
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return Player{}, false, fmt.Errorf("invalid player selection")
	}

	for _, player := range players {
		if player.ID == id {
			return player, true, nil
		}
	}

	return Player{}, false, fmt.Errorf("selected player does not exist")
}

func scoreFromForm(r *http.Request, field string) (int, error) {
	score, err := strconv.Atoi(strings.TrimSpace(r.FormValue(field)))
	if err != nil || score < 0 {
		return 0, fmt.Errorf("enter valid non-negative scores")
	}
	return score, nil
}

func renderMatchFormError(w http.ResponseWriter, r *http.Request, message string) {
	if r.Header.Get("HX-Request") != "true" {
		http.Error(w, message, http.StatusUnprocessableEntity)
		return
	}

	if err := MatchFormError(message).Render(r.Context(), w); err != nil {
		log.Printf("render match form error: %v", err)
	}
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
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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
			err := LeagueContent(players).Render(r.Context(), w)
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

		if err := LeagueContent(players).Render(r.Context(), w); err != nil {
			http.Error(w, "Could not render player list", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /matches", func(w http.ResponseWriter, r *http.Request) {
		team1Player1, _, err := playerFromForm(r, players, "team_1_player_1", true)
		if err != nil {
			renderMatchFormError(w, r, err.Error())
			return
		}
		team1Player2, hasTeam1Player2, err := playerFromForm(r, players, "team_1_player_2", false)
		if err != nil {
			renderMatchFormError(w, r, err.Error())
			return
		}
		team2Player1, _, err := playerFromForm(r, players, "team_2_player_1", true)
		if err != nil {
			renderMatchFormError(w, r, err.Error())
			return
		}
		team2Player2, hasTeam2Player2, err := playerFromForm(r, players, "team_2_player_2", false)
		if err != nil {
			renderMatchFormError(w, r, err.Error())
			return
		}

		team1Score, err := scoreFromForm(r, "team_1_score")
		if err != nil {
			renderMatchFormError(w, r, err.Error())
			return
		}
		team2Score, err := scoreFromForm(r, "team_2_score")
		if err != nil {
			renderMatchFormError(w, r, err.Error())
			return
		}

		match := Match{
			Team1Players: []Player{team1Player1},
			Team2Players: []Player{team2Player1},
			Team1Score:   team1Score,
			Team2Score:   team2Score,
		}
		if hasTeam1Player2 {
			match.Team1Players = append(match.Team1Players, team1Player2)
		}
		if hasTeam2Player2 {
			match.Team2Players = append(match.Team2Players, team2Player2)
		}

		match, err = store.addMatch(r.Context(), match)
		if err != nil {
			if errors.Is(err, ErrInvalidMatch) {
				renderMatchFormError(w, r, strings.TrimPrefix(err.Error(), ErrInvalidMatch.Error()+": "))
				return
			}

			http.Error(w, "Could not save match", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("X-Match-Added", "true")
			message := fmt.Sprintf("Match recorded: %d–%d", match.Team1Score, match.Team2Score)
			if err := MatchFormSuccess(message).Render(r.Context(), w); err != nil {
				http.Error(w, "Could not render match confirmation", http.StatusInternalServerError)
			}
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	log.Fatal(http.ListenAndServe(":8080", mux))

}
