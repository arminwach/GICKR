package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrInvalidMatch = errors.New("invalid match")

type Store struct {
	db *sql.DB
}

func newStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type Player struct {
	ID     int64
	Name   string
	Rating int
}

type Match struct {
	ID           int64
	Team1Players []Player
	Team2Players []Player
	Team1Score   int
	Team2Score   int
	PlayedAt     string
}

func validateMatch(match Match) error {
	if len(match.Team1Players) != len(match.Team2Players) || len(match.Team1Players) < 1 || len(match.Team1Players) > 2 {
		return fmt.Errorf("%w: teams must contain one or two players each", ErrInvalidMatch)
	}

	if match.Team1Score < 0 || match.Team2Score < 0 {
		return fmt.Errorf("%w: scores cannot be negative", ErrInvalidMatch)
	}

	if match.Team1Score == match.Team2Score {
		return fmt.Errorf("%w: matches cannot end in a draw", ErrInvalidMatch)
	}

	seenPlayers := make(map[int64]struct{})
	for _, team := range [][]Player{match.Team1Players, match.Team2Players} {
		for _, player := range team {
			if player.ID <= 0 {
				return fmt.Errorf("%w: every player must have a valid ID", ErrInvalidMatch)
			}

			if _, exists := seenPlayers[player.ID]; exists {
				return fmt.Errorf("%w: a player cannot appear twice", ErrInvalidMatch)
			}

			seenPlayers[player.ID] = struct{}{}
		}
	}

	return nil
}

func (s *Store) createTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			rating INTEGER NOT NULL DEFAULT 1000
		);

		CREATE TABLE IF NOT EXISTS matches (
			id INTEGER PRIMARY KEY,
			team_1_score INTEGER NOT NULL CHECK (team_1_score >= 0),
			team_2_score INTEGER NOT NULL CHECK (team_2_score >= 0),
			played_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (team_1_score != team_2_score)
		);

		CREATE TABLE IF NOT EXISTS match_players (
			match_id INTEGER NOT NULL,
			player_id INTEGER NOT NULL,
			team INTEGER NOT NULL CHECK (team IN (1, 2)),
			PRIMARY KEY (match_id, player_id),
			FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE,
			FOREIGN KEY (player_id) REFERENCES players(id)
		);
	`)

	return err
}

func (s *Store) addPlayer(ctx context.Context, player Player) (Player, error) {
	result, err := s.db.ExecContext(
		ctx,
		"INSERT INTO players (name, rating) VALUES (?, ?)",
		player.Name,
		player.Rating,
	)
	if err != nil {
		return Player{}, err
	}

	player.ID, err = result.LastInsertId()
	if err != nil {
		return Player{}, err
	}

	return player, nil
}

func (s *Store) addMatch(ctx context.Context, match Match) (Match, error) {
	if err := validateMatch(match); err != nil {
		return Match{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Match{}, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(
		ctx,
		"INSERT INTO matches (team_1_score, team_2_score) VALUES (?, ?)",
		match.Team1Score,
		match.Team2Score,
	)
	if err != nil {
		return Match{}, err
	}

	match.ID, err = result.LastInsertId()
	if err != nil {
		return Match{}, err
	}

	insertPlayer, err := tx.PrepareContext(
		ctx,
		"INSERT INTO match_players (match_id, player_id, team) VALUES (?, ?, ?)",
	)
	if err != nil {
		return Match{}, err
	}
	defer insertPlayer.Close()

	for teamNumber, players := range map[int][]Player{
		1: match.Team1Players,
		2: match.Team2Players,
	} {
		for _, player := range players {
			if _, err := insertPlayer.ExecContext(ctx, match.ID, player.ID, teamNumber); err != nil {
				return Match{}, err
			}
		}
	}

	if err := tx.QueryRowContext(
		ctx,
		"SELECT played_at FROM matches WHERE id = ?",
		match.ID,
	).Scan(&match.PlayedAt); err != nil {
		return Match{}, err
	}

	if err := tx.Commit(); err != nil {
		return Match{}, err
	}

	return match, nil
}

func (s *Store) deletePlayer(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(
		ctx,
		"DELETE FROM players WHERE id = ?",
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Store) loadPlayers() ([]Player, error) {
	rows, err := s.db.Query(`
		SELECT id, name, rating
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

		if err := rows.Scan(&player.ID, &player.Name, &player.Rating); err != nil {
			return nil, err
		}

		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return players, nil
}
