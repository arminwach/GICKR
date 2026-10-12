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
	ID     int64
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
