package main

import (
	"context"
	"database/sql"
)

type Store struct {
	db *sql.DB
}

func newStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type Player struct {
	Name   string
	Rating int
}

func (s *Store) createTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			rating INTEGER NOT NULL DEFAULT 1000
		)
	`)

	return err
}

func (s *Store) addPlayer(ctx context.Context, player Player) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO players (name, rating) VALUES (?, ?)",
		player.Name,
		player.Rating,
	)

	return err
}

func (s *Store) loadPlayers() ([]Player, error) {
	rows, err := s.db.Query(`
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
